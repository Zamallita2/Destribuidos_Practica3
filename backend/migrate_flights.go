package main

import (
	"log"

	"airres-api/db"
	"airres-api/models"
	"gorm.io/gorm"
)

// reconcileLegacyFlights removes unbooked copies made by the former import,
// which wrote every CSV row to both PostgreSQL nodes. Booked copies are kept
// for manual migration so a passenger record is never silently discarded.
func reconcileLegacyFlights() {
	if !db.IsAvailable(db.PGAmerica) || !db.IsAvailable(db.PGEuropaAsia) {
		return
	}
	marker := models.MigrationMarker{Key: "legacy_region_split_v1"}
	var amCount, euCount int64
	db.PGAmerica.Model(&models.MigrationMarker{}).Where("key = ?", marker.Key).Count(&amCount)
	db.PGEuropaAsia.Model(&models.MigrationMarker{}).Where("key = ?", marker.Key).Count(&euCount)
	if amCount > 0 && euCount > 0 {
		return
	}
	failed := false
	for _, item := range []struct {
		conn          *gorm.DB
		removeAmerica bool
		name          string
	}{
		{db.PGAmerica, false, "America"},
		{db.PGEuropaAsia, true, "EuropaAsia"},
	} {
		if item.conn == nil {
			continue
		}
		comparison := "<>"
		if item.removeAmerica {
			comparison = "="
		}
		condition := "EXISTS (SELECT 1 FROM ciudades c WHERE c.id = vuelos.id_origen AND c.region " + comparison + " 'America')"
		result := item.conn.Exec("DELETE FROM vuelos WHERE " + condition + " AND NOT EXISTS (SELECT 1 FROM boletos b WHERE b.id_vuelo = vuelos.id)")
		if result.Error != nil {
			failed = true
			log.Printf("[Migration] %s cleanup failed: %v", item.name, result.Error)
		} else {
			log.Printf("[Migration] %s removed %d unbooked non-owner copies", item.name, result.RowsAffected)
		}
		var remaining int64
		item.conn.Raw("SELECT COUNT(*) FROM vuelos WHERE " + condition).Scan(&remaining)
		if remaining > 0 {
			log.Printf("[Migration] %s has %d booked legacy flights requiring manual relocation", item.name, remaining)
		}
	}
	// Legacy European flights used IDs below the region partition. Move them
	// and their tickets together so routing remains unambiguous.
	if db.PGEuropaAsia != nil {
		err := db.PGEuropaAsia.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("UPDATE boletos SET id_vuelo = id_vuelo + 1000000000 WHERE id_vuelo < 1000000000 AND EXISTS (SELECT 1 FROM vuelos v JOIN ciudades c ON c.id = v.id_origen WHERE v.id = boletos.id_vuelo AND c.region <> 'America')").Error; err != nil {
				return err
			}
			if err := tx.Exec("UPDATE vuelos SET id = id + 1000000000 WHERE id < 1000000000 AND EXISTS (SELECT 1 FROM ciudades c WHERE c.id = vuelos.id_origen AND c.region <> 'America')").Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			failed = true
			log.Printf("[Migration] Europe flight ID migration failed: %v", err)
		}
	}
	if !failed {
		db.PGAmerica.Create(&marker)
		db.PGEuropaAsia.Create(&marker)
	}
}
