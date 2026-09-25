package db

import (
	"log"

	"gorm.io/gorm"
)

func EnsureBookingConstraints(database *gorm.DB) {
	if database == nil {
		return
	}
	const index = `CREATE UNIQUE INDEX IF NOT EXISTS unique_active_seat
		ON boletos (id_vuelo, id_asiento)
		WHERE estado IN ('RESERVED', 'SALED', 'REFUNDED')`
	if err := database.Exec(index).Error; err != nil {
		log.Fatalf("Cannot enforce unique active reservations: %v", err)
	}
}
