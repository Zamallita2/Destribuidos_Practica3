package services

import (
	"context"
	"log"
	"strings"
	"time"

	"airres-api/db"
	"airres-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

// SyncEvent represents a database action to be synchronized
type SyncEvent struct {
	Action       string      // CREATE, UPDATE, DELETE
	Entity       string      // avion, ciudad, vuelo, etc.
	Data         interface{} // The struct itself
	LamportClock int64       // Lamport timestamp for conflict resolution
}

var SyncChannel = make(chan SyncEvent, 100)

// StartSyncService runs in a separate goroutine and listens for changes
func StartSyncService() {
	log.Println("[Sync Service] Started Multi-Master Syncing Goroutine")

	for event := range SyncChannel {
		log.Printf("[Sync Service] Received %s for %s\n", event.Action, event.Entity)

		// Analyze special reimbursement logic
		if event.Entity == "Asiento" {
			asiento, ok := event.Data.(*models.Asiento)
			// Assuming we track the previous state or we just delay ANY transition TO AVAILABLE that implies a refund
			// For this simulation, if we receive an AVAILABLE seat event, we assume it's a refund and wait 15 sec
			if ok && asiento.Estado == "AVAILABLE" {
				log.Println("[Sync Service] Latency detected: Reembolso de tarjeta de crédito en proceso. Esperando 15s...")
				time.Sleep(15 * time.Second)
				log.Println("[Sync Service] Reembolso confirmado. Propagando actualización AVAILABLE de asiento...")
			}
		}

		// 1. Sync to Postgres America (if not origin)
		if db.PGAmerica != nil {
			syncToPostgres(db.PGAmerica, event, "America")
		}

		// 2. Sync to Postgres Europa (if not origin)
		if db.PGEuropaAsia != nil {
			syncToPostgres(db.PGEuropaAsia, event, "Europa/Asia")
		}

		// 3. Sync to MongoDB Backup
		if db.MongoClient != nil {
			syncToMongo(event)
		}
	}
}

// syncToPostgres performs an Upsert logic or Save logic based on the action
func syncToPostgres(pgDb *gorm.DB, event SyncEvent, node string) {
	GlobalLamportClock.UpdateClock(event.LamportClock)

	// GORM's Save performs an UPSERT (updates if exists, creates if not)
	if event.Action == "CREATE" || event.Action == "UPDATE" {
		incomingClock := event.LamportClock
		id := getIDFromData(event.Data)
		
		if incomingClock > 0 && id != nil {
			var existingClock int64 = -1
			switch event.Entity {
			case "Vuelo":
				var v models.Vuelo
				if pgDb.First(&v, id).Error == nil { existingClock = v.LamportClock }
			case "Boleto":
				var b models.Boleto
				if pgDb.First(&b, id).Error == nil { existingClock = b.LamportClock }
			case "Asiento":
				var a models.Asiento
				if pgDb.First(&a, id).Error == nil { existingClock = a.LamportClock }
			}
			
			if existingClock >= incomingClock {
				log.Printf("[Sync Service %s] Lamport Conflict Resolved: Rejected stale %s update (Incoming: %d, Existing: %d)\n", node, event.Entity, incomingClock, existingClock)
				return
			}
		}

		err := pgDb.Save(event.Data).Error
		if err != nil {
			log.Printf("[Sync Service] Error syncing to DB %s: %v\n", node, err)
			return
		}
		log.Printf("[Sync Service] Successfully synced %s to %s DB\n", event.Entity, node)
	} else if event.Action == "DELETE" {
		err := pgDb.Delete(event.Data).Error
		if err != nil {
			log.Printf("[Sync Service] Error deleting from DB %s: %v\n", node, err)
		}
	}
}

// syncToMongo writes data to the MongoDB instance
func syncToMongo(event SyncEvent) {
	if db.MongoDatabase == nil {
		return
	}

	collectionName := ""
	switch event.Entity {
	case "Vuelo":   collectionName = "vuelos"
	case "Boleto":  collectionName = "boletos"
	case "Asiento": collectionName = "asientos"
	case "Avion":   collectionName = "aviones"
	case "Ciudad":  collectionName = "ciudades"
	case "Puerta":  collectionName = "puertas"
	case "Precios": collectionName = "precios"
	default:        collectionName = strings.ToLower(event.Entity)
	}

	coll := db.MongoDatabase.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	GlobalLamportClock.UpdateClock(event.LamportClock)

	if event.Action == "CREATE" || event.Action == "UPDATE" {
		var filter bson.M

		// For Asiento, we use codigo+id_avion as secondary key for robustness, but we still ensure 'id' is set.
		if event.Entity == "Asiento" {
			if a, ok := event.Data.(*models.Asiento); ok {
				filter = bson.M{"codigo": a.Codigo, "id_avion": a.IDAvion}
				update := bson.M{"$set": bson.M{
					"id":       a.ID,
					"codigo":   a.Codigo,
					"id_avion": a.IDAvion,
					"estado":   a.Estado,
					"clase":    a.Clase,
				}}
				opts := options.Update().SetUpsert(true)
				_, err := coll.UpdateOne(ctx, filter, update, opts)
				if err != nil {
					log.Printf("[Sync Service] Error syncing Asiento to Mongo: %v\n", err)
				} else {
					log.Printf("[Sync Service] Synced Asiento %d (%s) -> %s to MongoDB\n", a.ID, a.Codigo, a.Estado)
				}
				return
			}
		}

		// For Boleto use id_boleto, for everything else use id
		if event.Entity == "Boleto" {
			filter = bson.M{"id_boleto": getIDFromData(event.Data)}
		} else {
			filter = bson.M{"id": getIDFromData(event.Data)}
		}

		// Lamport conflict resolution for MongoDB
		incomingClock := event.LamportClock
		if incomingClock > 0 {
			var result bson.M
			if err := coll.FindOne(ctx, filter).Decode(&result); err == nil {
				var existingClock int64 = -1
				if lc, ok := result["lamport_clock"].(int64); ok {
					existingClock = lc
				} else if lc, ok := result["lamport_clock"].(int32); ok {
					existingClock = int64(lc)
				}
				if existingClock >= incomingClock {
					log.Printf("[Sync Service Mongo] Lamport Conflict Resolved: Rejected stale %s update (Incoming: %d, Existing: %d)\n", event.Entity, incomingClock, existingClock)
					return
				}
			}
		}

		opts := options.Replace().SetUpsert(true)
		_, err := coll.ReplaceOne(ctx, filter, event.Data, opts)
		if err != nil {
			log.Printf("[Sync Service] Error syncing to Mongo: %v\n", err)
			return
		}
		log.Printf("[Sync Service] Successfully synced %s to MongoDB Asia\n", event.Entity)
	} else if event.Action == "DELETE" {
		filter := bson.M{"id": getIDFromData(event.Data)}
		coll.DeleteOne(ctx, filter)
	}
}

// getIDFromData is a helper to extract the ID field from various models via reflection or type assertion
func getIDFromData(data interface{}) interface{} {
	// Simple type switch for our known models to avoid heavy reflection
	switch v := data.(type) {
	case *models.Vuelo: return v.ID
	case *models.Boleto: return v.IDBoleto
	case *models.Asiento: return v.ID
	case *models.Avion: return v.ID
	case *models.Ciudad: return v.ID
	case *models.Puerta: return v.ID
	}
	return nil
}

// getLamportClockFromData is a helper to extract the LamportClock from known models
func getLamportClockFromData(data interface{}) int64 {
	switch v := data.(type) {
	case *models.Vuelo: return v.LamportClock
	case *models.Boleto: return v.LamportClock
	case *models.Asiento: return v.LamportClock
	}
	return 0
}

// SendSyncEvent pushes a new event into the background channel
func SendSyncEvent(action, entity string, data interface{}, clock int64) {
	// Non-blocking send
	select {
	case SyncChannel <- SyncEvent{Action: action, Entity: entity, Data: data, LamportClock: clock}:
	default:
		log.Println("[Sync Service] WARNING: Sync channel full, event dropped")
	}
}
