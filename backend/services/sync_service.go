package services

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"airres-api/db"
	"airres-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

// SyncEvent represents a database action to be synchronized
type SyncEvent struct {
	Action       string      // CREATE, UPDATE, DELETE
	Entity       string      // avion, ciudad, vuelo, etc.
	Data         interface{} // The struct itself
	LamportClock int64       // Lamport timestamp for conflict resolution
	VectorClock  string      // Vector clock for advanced conflict resolution
	NodeID       string
}

var SyncChannel = make(chan SyncEvent, 100)

// StartSyncService runs in a separate goroutine and listens for changes
func StartSyncService() {
	log.Println("[Sync Service] Started Multi-Master Syncing Goroutine")

	for event := range SyncChannel {
		log.Printf("[Sync Service] Received %s for %s\n", event.Action, event.Entity)
		if err := ApplySyncEvent(event); err != nil {
			log.Printf("[Sync Service] delivery failed: %v", err)
		}
	}
}

func ApplySyncEvent(event SyncEvent) error {
	// The two regional PostgreSQL servers also hold read/failover replicas.
	// Try every node independently, so a failed node cannot delay a live one.
	var failures []error
	if db.PGAmerica == nil {
		failures = append(failures, errors.New("postgres_america_unavailable"))
	} else if err := syncToPostgres(db.PGAmerica, event, "America"); err != nil {
		failures = append(failures, err)
	}
	if db.PGEuropaAsia == nil {
		failures = append(failures, errors.New("postgres_europa_unavailable"))
	} else if err := syncToPostgres(db.PGEuropaAsia, event, "Europa/Asia"); err != nil {
		failures = append(failures, err)
	}
	if db.MongoDatabase == nil {
		failures = append(failures, errors.New("mongodb_unavailable"))
	} else if err := syncToMongo(event); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

// syncToPostgres performs an Upsert logic or Save logic based on the action
func syncToPostgres(pgDb *gorm.DB, event SyncEvent, node string) error {
	GlobalLamportClock.UpdateClock(event.LamportClock)

	// GORM's Save performs an UPSERT (updates if exists, creates if not)
	if event.Action == "CREATE" || event.Action == "UPDATE" {
		UpdateVectorClock(event.VectorClock)
		incomingClock := event.LamportClock
		incomingVector := event.VectorClock
		id := getIDFromData(event.Data)

		if incomingClock > 0 && id != nil {
			var existingClock int64 = -1
			var existingVector string = ""
			var existingNode string
			switch event.Entity {
			case "Vuelo":
				var v models.Vuelo
				if pgDb.First(&v, id).Error == nil {
					existingClock = v.LamportClock
					existingVector = v.VectorClock
					existingNode = v.SourceNode
				}
			case "Boleto":
				var b models.Boleto
				if pgDb.First(&b, id).Error == nil {
					existingClock = b.LamportClock
					existingVector = b.VectorClock
					existingNode = b.SourceNode
				}
			case "Asiento":
				var a models.Asiento
				if pgDb.First(&a, id).Error == nil {
					existingClock = a.LamportClock
					existingVector = a.VectorClock
					existingNode = a.SourceNode
				}
			case "OcupacionVuelo":
				var occupancy models.OcupacionVuelo
				if pgDb.First(&occupancy, "id_vuelo = ?", id).Error == nil {
					existingClock = occupancy.LamportClock
					existingVector = occupancy.VectorClock
					existingNode = occupancy.SourceNode
				}
			}
			if !ShouldApplyVersion(incomingVector, existingVector, incomingClock, existingClock, event.NodeID, existingNode) {
				log.Printf("[Sync Service %s] Conflict Resolved: Rejected stale %s update\n", node, event.Entity)
				return nil
			}
		}

		err := pgDb.Save(event.Data).Error
		if err != nil {
			log.Printf("[Sync Service] Error syncing to DB %s: %v\n", node, err)
			return err
		}
		log.Printf("[Sync Service] Successfully synced %s to %s DB\n", event.Entity, node)
	} else if event.Action == "DELETE" {
		err := pgDb.Delete(event.Data).Error
		if err != nil {
			log.Printf("[Sync Service] Error deleting from DB %s: %v\n", node, err)
			return err
		}
	}
	return nil
}

// syncToMongo writes data to the MongoDB instance
func syncToMongo(event SyncEvent) error {
	if db.MongoDatabase == nil {
		return errors.New("mongodb_unavailable")
	}

	collectionName := ""
	switch event.Entity {
	case "Vuelo":
		collectionName = "vuelos"
	case "Boleto":
		collectionName = "boletos"
	case "Asiento":
		collectionName = "asientos"
	case "OcupacionVuelo":
		collectionName = "ocupaciones_vuelo"
	case "Avion":
		collectionName = "aviones"
	case "Ciudad":
		collectionName = "ciudades"
	case "Puerta":
		collectionName = "puertas"
	case "Precios":
		collectionName = "precios"
	default:
		collectionName = strings.ToLower(event.Entity)
	}

	coll := db.MongoDatabase.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	GlobalLamportClock.UpdateClock(event.LamportClock)

	if event.Action == "CREATE" || event.Action == "UPDATE" {
		UpdateVectorClock(event.VectorClock)
		incomingClock := event.LamportClock
		incomingVector := event.VectorClock
		id := getIDFromData(event.Data)

		var filter bson.M
		if event.Entity == "Boleto" {
			filter = bson.M{"id_boleto": id}
		} else if event.Entity == "OcupacionVuelo" {
			filter = bson.M{"id_vuelo": id}
		} else {
			filter = bson.M{"id": id}
		}

		if incomingClock > 0 && id != nil {
			var existingClock int64 = -1
			var existingVector string = ""
			var existingNode string

			var result map[string]interface{}
			if err := coll.FindOne(ctx, filter).Decode(&result); err == nil {
				if val, ok := result["lamport_clock"]; ok {
					switch v := val.(type) {
					case int32:
						existingClock = int64(v)
					case int64:
						existingClock = v
					case float64:
						existingClock = int64(v)
					}
				}
				if val, ok := result["vector_clock"]; ok {
					if v, isStr := val.(string); isStr {
						existingVector = v
					}
				}
				if node, ok := result["source_node"].(string); ok {
					existingNode = node
				}
			} else if err != mongo.ErrNoDocuments {
				return err
			}
			if !ShouldApplyVersion(incomingVector, existingVector, incomingClock, existingClock, event.NodeID, existingNode) {
				log.Printf("[Sync Service Mongo] Conflict Resolved: Rejected stale %s update\n", event.Entity)
				return nil
			}
		}

		opts := options.Replace().SetUpsert(true)
		_, err := coll.ReplaceOne(ctx, filter, event.Data, opts)
		if err != nil {
			log.Printf("[Sync Service] Error syncing to Mongo: %v\n", err)
			return err
		}
		log.Printf("[Sync Service] Successfully synced %s to MongoDB Asia\n", event.Entity)
	} else if event.Action == "DELETE" {
		var filter bson.M
		if event.Entity == "Boleto" {
			filter = bson.M{"id_boleto": getIDFromData(event.Data)}
		} else {
			filter = bson.M{"id": getIDFromData(event.Data)}
		}
		_, err := coll.DeleteOne(ctx, filter)
		return err
	}
	return nil
}

// getIDFromData is a helper to extract the ID field from various models via reflection or type assertion
func getIDFromData(data interface{}) interface{} {
	// Simple type switch for our known models to avoid heavy reflection
	switch v := data.(type) {
	case *models.Vuelo:
		return v.ID
	case *models.Boleto:
		return v.IDBoleto
	case *models.Asiento:
		return v.ID
	case *models.OcupacionVuelo:
		return v.IDVuelo
	case *models.Avion:
		return v.ID
	case *models.Ciudad:
		return v.ID
	case *models.Puerta:
		return v.ID
	}
	return nil
}

// getLamportClockFromData is a helper to extract the LamportClock from known models
func getLamportClockFromData(data interface{}) int64 {
	switch v := data.(type) {
	case *models.Vuelo:
		return v.LamportClock
	case *models.Boleto:
		return v.LamportClock
	case *models.Asiento:
		return v.LamportClock
	}
	return 0
}

// SendSyncEvent pushes a new event into the background channel
func SendSyncEvent(action, entity string, data interface{}, clock int64, vclock string) {
	SyncChannel <- SyncEvent{Action: action, Entity: entity, Data: data, LamportClock: clock, VectorClock: vclock, NodeID: NodeID}
}
