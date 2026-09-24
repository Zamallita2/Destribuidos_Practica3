package services

import (
	"log"

	"airres-api/db"
	"airres-api/models"
	"gorm.io/gorm"
)

// RestoreClocks carries logical time across process restarts. The outbox is
// durable and contains the vector attached to each locally committed event.
func RestoreClocks() {
	for _, conn := range []*gorm.DB{db.PGAmerica, db.PGEuropaAsia} {
		if conn == nil {
			continue
		}
		var rows []models.SyncOutbox
		if err := conn.Select("lamport_clock", "vector_clock").Find(&rows).Error; err != nil {
			log.Printf("[Clock] restore failed: %v", err)
			continue
		}
		for _, row := range rows {
			if row.LamportClock > GlobalLamportClock.GetCurrent() {
				GlobalLamportClock.UpdateClock(row.LamportClock)
			}
			UpdateVectorClock(row.VectorClock)
		}
	}
}
