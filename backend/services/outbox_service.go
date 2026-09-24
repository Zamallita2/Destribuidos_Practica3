package services

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"airres-api/db"
	"airres-api/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func EncodeOutboxEvent(action, entity string, data interface{}) (models.SyncOutbox, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return models.SyncOutbox{}, err
	}
	row := models.SyncOutbox{EventID: uuid.NewString(), Action: action, Entity: entity,
		Payload: payload, CreatedAt: time.Now().UTC()}
	switch value := data.(type) {
	case *models.Boleto:
		row.LamportClock, row.VectorClock, row.NodeID = value.LamportClock, value.VectorClock, value.SourceNode
	case *models.Vuelo:
		row.LamportClock, row.VectorClock, row.NodeID = value.LamportClock, value.VectorClock, value.SourceNode
	case *models.Asiento:
		row.LamportClock, row.VectorClock, row.NodeID = value.LamportClock, value.VectorClock, value.SourceNode
	case *models.OcupacionVuelo:
		row.LamportClock, row.VectorClock, row.NodeID = value.LamportClock, value.VectorClock, value.SourceNode
	default:
		return models.SyncOutbox{}, errors.New("entidad_no_sincronizable")
	}
	return row, nil
}

func DecodeOutboxEvent(row models.SyncOutbox) (SyncEvent, error) {
	event := SyncEvent{Action: row.Action, Entity: row.Entity,
		LamportClock: row.LamportClock, VectorClock: row.VectorClock, NodeID: row.NodeID}
	switch row.Entity {
	case "Boleto":
		event.Data = &models.Boleto{}
	case "Vuelo":
		event.Data = &models.Vuelo{}
	case "Asiento":
		event.Data = &models.Asiento{}
	case "OcupacionVuelo":
		event.Data = &models.OcupacionVuelo{}
	default:
		return SyncEvent{}, errors.New("entidad_no_sincronizable")
	}
	if err := json.Unmarshal(row.Payload, event.Data); err != nil {
		return SyncEvent{}, err
	}
	return event, nil
}

func QueueOutboxEvent(tx *gorm.DB, action, entity string, data interface{}) error {
	row, err := EncodeOutboxEvent(action, entity, data)
	if err != nil {
		return err
	}
	return tx.Create(&row).Error
}

// StartOutboxWorker retries every committed event until both relational
// replicas and the MongoDB projection acknowledge it.
func StartOutboxWorker() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		drainOutbox(db.PGAmerica)
		drainOutbox(db.PGEuropaAsia)
	}
}

func drainOutbox(conn *gorm.DB) {
	if !db.IsAvailable(conn) {
		return
	}
	for i := 0; i < 500; i++ {
		found := false
		err := conn.Transaction(func(tx *gorm.DB) error {
			var row models.SyncOutbox
			result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("delivered_at = 0 AND next_attempt_at <= ?", time.Now().Unix()).
				Order("created_at, event_id").Limit(1).Find(&row)
			if result.RowsAffected == 0 {
				return nil
			}
			if result.Error != nil {
				return result.Error
			}
			found = true
			event, decodeErr := DecodeOutboxEvent(row)
			if decodeErr == nil {
				decodeErr = ApplySyncEvent(event)
			}
			if decodeErr != nil {
				row.Attempts++
				row.LastError = decodeErr.Error()
				delay := row.Attempts * 2
				if delay > 60 {
					delay = 60
				}
				row.NextAttemptAt = time.Now().Add(time.Duration(delay) * time.Second).Unix()
			} else {
				row.DeliveredAt = time.Now().Unix()
				row.LastError = ""
			}
			return tx.Save(&row).Error
		})
		if err != nil {
			log.Printf("[Outbox] processing failed: %v", err)
			return
		}
		if !found {
			return
		}
	}
}
