package main

import (
	"log"
	"time"

	"airres-api/db"
	"airres-api/models"
	"gorm.io/gorm"
)

func migrateRecoveredNode(conn *gorm.DB) {
	if err := conn.AutoMigrate(&models.Avion{}, &models.Ciudad{}, &models.Puerta{}, &models.Asiento{}, &models.EstadoVuelo{}, &models.Vuelo{}, &models.Boleto{}, &models.Precios{}, &models.DetallesVuelos{}, &models.SyncOutbox{}, &models.OcupacionVuelo{}, &models.MigrationMarker{}, &models.IDAllocator{}); err != nil {
		log.Printf("[Recovery] migration failed: %v", err)
		return
	}
	db.EnsureBookingConstraints(conn)
}

func startRecoveryMonitor() {
	amUp, euUp, mongoUp := db.IsAvailable(db.PGAmerica), db.IsAvailable(db.PGEuropaAsia), db.IsMongoAvailable()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		amNow, euNow, mongoNow := db.IsAvailable(db.PGAmerica), db.IsAvailable(db.PGEuropaAsia), db.IsMongoAvailable()
		if amNow && !amUp {
			log.Println("[Recovery] PostgreSQL America restored")
			recordSyncEvent("reconciliation", "PG_AM recuperado; iniciando reconciliación")
			migrateRecoveredNode(db.PGAmerica)
			seedMatrices(db.PGAmerica)
			seedPrecios(db.PGAmerica)
			seedAsientos(db.PGAmerica)
		}
		if euNow && !euUp {
			log.Println("[Recovery] PostgreSQL Europa/Asia restored")
			recordSyncEvent("reconciliation", "PG_EU recuperado; iniciando reconciliación")
			migrateRecoveredNode(db.PGEuropaAsia)
			seedMatrices(db.PGEuropaAsia)
			seedPrecios(db.PGEuropaAsia)
			seedAsientos(db.PGEuropaAsia)
		}
		if (amNow && !amUp) || (euNow && !euUp) {
			seedAirportCatalog()
			reconcilePostgresReplicas()
			go seedMissingOccupancy()
		}
		if mongoNow && !mongoUp {
			log.Println("[Recovery] MongoDB restored")
			recordSyncEvent("reconciliation", "MongoDB recuperado; reconstruyendo snapshot")
			seedMongoMatrices()
			bootstrapMongo(false)
		}
		amUp, euUp, mongoUp = amNow, euNow, mongoNow
	}
}
