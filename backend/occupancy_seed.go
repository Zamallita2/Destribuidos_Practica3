package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"airres-api/db"
	"airres-api/models"
	"airres-api/services"
	"gorm.io/gorm"
)

// seedMissingOccupancy resumes from PostgreSQL after a restart. Booking and
// seat reads also create the manifest on demand, so those flights are usable
// before the background sweep reaches them.
func seedMissingOccupancy() {
	maxFlights, _ := strconv.Atoi(os.Getenv("SEED_OCCUPANCY_MAX_FLIGHTS"))
	processed := 0
	for _, conn := range []*gorm.DB{db.PGAmerica, db.PGEuropaAsia} {
		if !db.IsAvailable(conn) {
			continue
		}
		lastID := uint(0)
		rangeCondition := ""
		if conn == db.PGAmerica && db.IsAvailable(db.PGEuropaAsia) {
			rangeCondition = " AND v.id < 1000000000"
		}
		if conn == db.PGEuropaAsia && db.IsAvailable(db.PGAmerica) {
			rangeCondition = " AND v.id >= 1000000000"
		}
		for {
			var flights []models.Vuelo
			err := conn.Raw(`SELECT v.* FROM vuelos v LEFT JOIN ocupaciones_vuelo o ON o.id_vuelo = v.id
				WHERE v.id > ? AND (o.id_vuelo IS NULL OR o.matrix_hash <> ?)`+rangeCondition+` ORDER BY v.id LIMIT 100`, lastID, services.CurrentMatrixHash()).Scan(&flights).Error
			if err != nil {
				log.Printf("[Occupancy] scan failed: %v", err)
				break
			}
			if len(flights) == 0 {
				break
			}
			for _, flight := range flights {
				lastID = flight.ID
				err = conn.Transaction(func(tx *gorm.DB) error { _, err := services.EnsureFlightOccupancy(tx, flight); return err })
				if err != nil {
					log.Printf("[Occupancy] flight %d failed: %v", flight.ID, err)
				} else {
					processed++
				}
				if maxFlights > 0 && processed >= maxFlights {
					log.Printf("[Occupancy] test limit reached: %d", processed)
					return
				}
				if processed%500 == 0 {
					log.Printf("[Occupancy] generated %d flight manifests", processed)
				}
			}
		}
	}
	log.Printf("[Occupancy] complete: generated %d flight manifests at %s", processed, time.Now().UTC().Format(time.RFC3339))
}
