package main

import (
	"fmt"
	"log"
	"time"

	"airres-api/db"
	"airres-api/models"
	"airres-api/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// reconcilePostgresReplicas fills missing records after startup or a node
// restart. Live updates are delivered through the durable outbox with clocks.
func reconcilePostgresReplicas() {
	if services.InputImportActive.Load() {
		return
	}
	if !db.IsAvailable(db.PGAmerica) || !db.IsAvailable(db.PGEuropaAsia) {
		return
	}
	var copiedFlights, copiedTickets int64
	for _, pair := range []struct {
		source, target *gorm.DB
		american       bool
	}{
		{db.PGAmerica, db.PGEuropaAsia, true},
		{db.PGEuropaAsia, db.PGAmerica, false},
	} {
		var lastFlight uint
		for {
			if services.InputImportActive.Load() {
				return
			}
			var flights []models.Vuelo
			query := pair.source.Where("id > ?", lastFlight)
			if pair.american {
				query = query.Where("id < ?", 1000000000)
			} else {
				query = query.Where("id >= ?", 1000000000)
			}
			if err := query.Order("id").Limit(200).Find(&flights).Error; err != nil {
				log.Printf("[PG Replica] flights read failed: %v", err)
				break
			}
			if len(flights) == 0 {
				break
			}
			result := pair.target.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&flights, 200)
			if err := result.Error; err != nil {
				log.Printf("[PG Replica] flights write failed: %v", err)
				break
			}
			copiedFlights += result.RowsAffected
			lastFlight = flights[len(flights)-1].ID
		}
		var lastTicket uint
		for {
			if services.InputImportActive.Load() {
				return
			}
			var tickets []models.Boleto
			query := pair.source.Where("id_boleto > ?", lastTicket)
			if pair.american {
				query = query.Where("id_boleto < ?", 1000000000)
			} else {
				query = query.Where("id_boleto >= ?", 1000000000)
			}
			if err := query.Order("id_boleto").Limit(200).Find(&tickets).Error; err != nil {
				log.Printf("[PG Replica] tickets read failed: %v", err)
				break
			}
			if len(tickets) == 0 {
				break
			}
			result := pair.target.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&tickets, 200)
			if err := result.Error; err != nil {
				log.Printf("[PG Replica] tickets write failed: %v", err)
				break
			}
			copiedTickets += result.RowsAffected
			lastTicket = tickets[len(tickets)-1].IDBoleto
		}
		var lastManifest uint
		for {
			if services.InputImportActive.Load() {
				return
			}
			var manifests []models.OcupacionVuelo
			query := pair.source.Where("id_vuelo > ?", lastManifest)
			if pair.american {
				query = query.Where("id_vuelo < ?", 1000000000)
			} else {
				query = query.Where("id_vuelo >= ?", 1000000000)
			}
			if err := query.Order("id_vuelo").Limit(50).Find(&manifests).Error; err != nil {
				log.Printf("[PG Replica] manifest read failed: %v", err)
				break
			}
			if len(manifests) == 0 {
				break
			}
			if err := pair.target.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&manifests, 50).Error; err != nil {
				log.Printf("[PG Replica] manifest write failed: %v", err)
				break
			}
			lastManifest = manifests[len(manifests)-1].IDVuelo
		}
	}
	if copiedFlights > 0 || copiedTickets > 0 {
		recordSyncEvent("reconciled", fmt.Sprintf("Reconciliación PG: %d vuelos y %d boletos copiados", copiedFlights, copiedTickets))
	}
}

func startReplicaReconciler() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		reconcilePostgresReplicas()
	}
}
