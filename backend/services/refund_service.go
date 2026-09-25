package services

import (
	"log"
	"os"
	"strconv"
	"time"

	"airres-api/db"
	"airres-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func RefundAvailableAt(now time.Time, delayMinutes int) int64 {
	if delayMinutes < 0 {
		delayMinutes = 15
	}
	return now.Add(time.Duration(delayMinutes) * time.Minute).Unix()
}

func ConfiguredRefundDelay() int {
	value := os.Getenv("REFUND_DELAY_MINUTES")
	if value == "" {
		return 15
	}
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes < 0 {
		return 15
	}
	return minutes
}

// StartRefundWorker persists the delay in boleto.available_at. Restarting
// the process does not reset the countdown or block unrelated sync events.
func StartRefundWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		releaseRefunds(db.PGAmerica)
		releaseRefunds(db.PGEuropaAsia)
	}
}

func releaseRefunds(conn *gorm.DB) {
	if !db.IsAvailable(conn) {
		return
	}
	var due []models.Boleto
	query := conn.Where("estado = ? AND available_at <= ?", "REFUNDED", time.Now().Unix())
	if conn == db.PGEuropaAsia && db.IsAvailable(db.PGAmerica) {
		query = query.Where("id_boleto >= ?", 1000000000)
	} else if conn == db.PGAmerica && db.IsAvailable(db.PGEuropaAsia) {
		query = query.Where("id_boleto < ?", 1000000000)
	}
	if err := query.Limit(100).Find(&due).Error; err != nil {
		log.Printf("[Refund] query failed: %v", err)
		return
	}
	for _, ticket := range due {
		var updated models.Boleto
		err := conn.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&updated, ticket.IDBoleto).Error; err != nil {
				return err
			}
			if updated.Estado != "REFUNDED" || updated.AvailableAt > time.Now().Unix() {
				return nil
			}
			updated.Estado = "ANNULLED"
			updated.LamportClock = GlobalLamportClock.Tick()
			updated.VectorClock = TickVectorClock()
			updated.SourceNode = NodeID
			if err := tx.Save(&updated).Error; err != nil {
				return err
			}
			return QueueOutboxEvent(tx, "UPDATE", "Boleto", &updated)
		})
		if err != nil {
			log.Printf("[Refund] release failed for %d: %v", ticket.IDBoleto, err)
		}
	}
}
