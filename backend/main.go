package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"airres-api/data"
	"airres-api/db"
	"airres-api/models"
	"airres-api/routes"
	"airres-api/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

func main() {
	// Initialize databases
	db.InitPostgres()
	db.InitMongoDB()

	// Perform AutoMigrate
	if db.IsAvailable(db.PGAmerica) {
		db.PGAmerica.AutoMigrate(&models.Avion{}, &models.Ciudad{}, &models.Puerta{}, &models.Asiento{}, &models.EstadoVuelo{}, &models.Vuelo{}, &models.Boleto{}, &models.Precios{}, &models.DetallesVuelos{}, &models.SyncOutbox{}, &models.OcupacionVuelo{}, &models.MigrationMarker{}, &models.IDAllocator{})
		db.EnsureBookingConstraints(db.PGAmerica)
	}
	if db.IsAvailable(db.PGEuropaAsia) {
		db.PGEuropaAsia.AutoMigrate(&models.Avion{}, &models.Ciudad{}, &models.Puerta{}, &models.Asiento{}, &models.EstadoVuelo{}, &models.Vuelo{}, &models.Boleto{}, &models.Precios{}, &models.DetallesVuelos{}, &models.SyncOutbox{}, &models.OcupacionVuelo{}, &models.MigrationMarker{}, &models.IDAllocator{})
		db.EnsureBookingConstraints(db.PGEuropaAsia)
	}
	services.RestoreClocks()
	seedAirportCatalog()

	// Seed Matrix Data
	seedMatrices(db.PGAmerica)
	seedMatrices(db.PGEuropaAsia)

	seedPrecios(db.PGAmerica)
	seedPrecios(db.PGEuropaAsia)

	seedAsientos(db.PGAmerica)
	seedAsientos(db.PGEuropaAsia)
	reconcileLegacyFlights()

	// Seed Flights from CSV dataset (auto-import, skips if already loaded)
	csvPath := data.CSVPath()
	if csvPath != "" {
		reportPath := os.Getenv("REJECTED_CSV_PATH")
		if reportPath == "" {
			reportPath = filepath.Join("reports", "vuelos_rechazados.csv")
		}
		if count, err := data.GenerateRejectedCSV(csvPath, filepath.Join("data", "matrices.json"), reportPath); err != nil {
			log.Printf("[Import] Cannot write rejection report: %v", err)
		} else {
			log.Printf("[Import] Rejection report: %s (%d rows)", reportPath, count)
		}
		if db.IsAvailable(db.PGAmerica) {
			data.SeedFlightsFromCSV(db.PGAmerica, csvPath, "America")
		}
		if db.IsAvailable(db.PGEuropaAsia) {
			data.SeedFlightsFromCSV(db.PGEuropaAsia, csvPath, "EuropaAsia")
		}
	} else {
		log.Println("[Seed Flights] WARNING: Dataset CSV not found. Place flights.csv in backend/data/ or dataset/ folder.")
	}
	seedDemoFlights()

	// Start Background Multi-Master Syncing Goroutine
	go services.StartOutboxWorker()
	go services.StartRefundWorker()
	go seedMissingOccupancy()
	go startReplicaReconciler()
	go startRecoveryMonitor()
	// A large existing dataset can take minutes to reconcile. Keep the API
	// available while the persisted outbox and snapshots catch up in the back.
	go func() {
		reconcilePostgresReplicas()
		seedMongoMatrices()
		bootstrapMongo(false)
		pruneMongoOrphans()
	}()

	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-User-Country", "X-Region"}
	r.Use(cors.New(config))

	// Register Routes
	routes.SetupRoutes(r)

	// Health check endpoint
	r.GET("/api/health", func(c *gin.Context) {
		country := c.GetHeader("X-User-Country")
		if country == "" {
			country = "Unknown"
		}
		am, eu, mongo := db.IsAvailable(db.PGAmerica), db.IsAvailable(db.PGEuropaAsia), db.IsMongoAvailable()
		status := "ok"
		if !am || !eu || !mongo {
			status = "degraded"
		}
		if !am && !eu {
			status = "unavailable"
		}
		c.JSON(200, gin.H{
			"status":  status,
			"nodes":   gin.H{"postgres_america": am, "postgres_europa_asia": eu, "mongodb": mongo},
			"message": "AirRes API is running",
			"country": country,
		})
	})

	log.Println("Starting server on :8080")
	r.Run(":8080")
}

func seedMongoMatrices() {
	if !db.IsMongoAvailable() {
		return
	}

	path := filepath.Join("data", "matrices.json")
	file, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[Seed Mongo] Warning: could not read %s: %v", path, err)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(file, &data); err != nil {
		log.Printf("[Seed Mongo] Error parsing JSON: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Upsert detalles_vuelos (travel time matrix)
	travelTime := data["travel_time"]
	_, err = db.MongoDatabase.Collection("detalles_vuelos").UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"matriz_tiempos": travelTime}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("[Seed Mongo] Error upserting detalles_vuelos: %v", err)
	} else {
		log.Println("[Seed Mongo] Successfully seeded detalles_vuelos into MongoDB")
	}

	// Upsert precios (economy + first class fares)
	_, err = db.MongoDatabase.Collection("precios").UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{
			"matriz_precios_regular": data["economy_fares"],
			"matriz_precios_vip":     data["first_class_fares"],
		}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("[Seed Mongo] Error upserting precios: %v", err)
	} else {
		log.Println("[Seed Mongo] Successfully seeded precios into MongoDB")
	}
}

func seedMatrices(dbConn *gorm.DB) {
	if !db.IsAvailable(dbConn) {
		return
	}

	path := filepath.Join("data", "matrices.json")
	file, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[Seed] Warning: could not find %s for seeding: %v", path, err)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(file, &data); err != nil {
		log.Printf("[Seed] Error parsing JSON: %v", err)
		return
	}

	travelTimesJSON, _ := json.Marshal(data["travel_time"])
	detalles := models.DetallesVuelos{
		MatrizTiempos: travelTimesJSON,
	}

	var existing models.DetallesVuelos
	if err := dbConn.First(&existing).Error; err == gorm.ErrRecordNotFound {
		if err := dbConn.Create(&detalles).Error; err != nil {
			log.Printf("[Seed] Error inserting details: %v", err)
		}
	} else if err != nil {
		log.Printf("[Seed] Error reading details: %v", err)
	} else if string(existing.MatrizTiempos) != string(travelTimesJSON) {
		if err := dbConn.Model(&existing).Update("matriz_tiempos", travelTimesJSON).Error; err != nil {
			log.Printf("[Seed] Error updating details: %v", err)
		}
	}
}

func seedPrecios(dbConn *gorm.DB) {
	if !db.IsAvailable(dbConn) {
		return
	}

	path := filepath.Join("data", "matrices.json")
	file, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[Seed] Error reading prices: %v", err)
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal(file, &data); err != nil {
		log.Printf("[Seed] Error parsing prices: %v", err)
		return
	}

	regJSON, _ := json.Marshal(data["economy_fares"])
	vipJSON, _ := json.Marshal(data["first_class_fares"])

	var existing models.Precios
	err = dbConn.First(&existing).Error
	if err != nil {
		// No row at all - create
		precios := models.Precios{MatrizPreciosRegular: regJSON, MatrizPreciosVip: vipJSON}
		dbConn.Create(&precios)
		log.Printf("[Seed] Created Precios matrix into %s", dbConn.Name())
	} else if string(existing.MatrizPreciosRegular) != string(regJSON) || string(existing.MatrizPreciosVip) != string(vipJSON) {
		dbConn.Model(&existing).Updates(map[string]interface{}{
			"matriz_precios_regular": regJSON,
			"matriz_precios_vip":     vipJSON,
		})
		log.Printf("[Seed] Updated Precios matrix into %s", dbConn.Name())
	} else {
		log.Printf("[Seed] Precios already seeded in %s, skipping.", dbConn.Name())
	}
}

func seedAsientos(dbConn *gorm.DB) {
	if !db.IsAvailable(dbConn) {
		return
	}
	var count int64
	dbConn.Model(&models.Asiento{}).Count(&count)
	if count > 0 {
		dbConn.Model(&models.Asiento{}).Where("estado <> ?", "AVAILABLE").Update("estado", "AVAILABLE")
		return
	}

	var aviones []models.Avion
	dbConn.Order("id").Find(&aviones)

	for _, avion := range aviones {
		log.Printf("[Seed] Generating seats for %s (VIP: %d, Regular: %d)", avion.Nombre, avion.AsientosVip, avion.AsientosRegular)
		seats := make([]models.Asiento, 0, avion.AsientosVip+avion.AsientosRegular)

		// Generate VIP seats (Row 1 to X)
		vipRows := (avion.AsientosVip / 4) + 1
		for i := 0; i < avion.AsientosVip; i++ {
			row := (i / 4) + 1
			col := string(rune('A' + (i % 4)))
			seat := models.Asiento{
				Codigo:  strconv.Itoa(row) + col,
				IDAvion: avion.ID,
				Estado:  "AVAILABLE",
				Clase:   "VIP",
			}
			seats = append(seats, seat)
		}

		// Generate Regular seats (Starting after VIP rows)
		for i := 0; i < avion.AsientosRegular; i++ {
			row := vipRows + (i / 6) + 1
			col := string(rune('A' + (i % 6)))
			seat := models.Asiento{
				Codigo:  strconv.Itoa(row) + col,
				IDAvion: avion.ID,
				Estado:  "AVAILABLE",
				Clase:   "REGULAR",
			}
			seats = append(seats, seat)
		}
		if err := dbConn.CreateInBatches(&seats, 500).Error; err != nil {
			log.Printf("[Seed] Could not seed seats for aircraft %d: %v", avion.ID, err)
			return
		}
	}
	log.Printf("[Seed] Successfully seeded Asientos into %s", dbConn.Name())
}

func syncPGSeatsToMongo() {
	if db.MongoClient == nil || db.PGAmerica == nil {
		return
	}
	var seats []models.Asiento
	db.PGAmerica.Find(&seats)

	if len(seats) == 0 {
		return
	}

	coll := db.MongoDatabase.Collection("asientos")
	ctx := context.Background()

	// Check if already synced
	count, _ := coll.CountDocuments(ctx, bson.M{})
	if count >= int64(len(seats)) {
		log.Println("[Seed Mongo] Asientos already synced in Mongo, skipping bulk sync.")
		return
	}

	log.Printf("[Seed Mongo] Syncing %d seats from PG to Mongo with bulk upsert...", len(seats))

	var models []mongo.WriteModel
	for _, seat := range seats {
		filter := bson.M{"codigo": seat.Codigo, "id_avion": seat.IDAvion}
		update := bson.M{"$set": bson.M{
			"id":       seat.ID,
			"codigo":   seat.Codigo,
			"id_avion": seat.IDAvion,
			"estado":   seat.Estado,
			"clase":    seat.Clase,
		}}
		model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, model)
	}

	if len(models) > 0 {
		_, err := coll.BulkWrite(ctx, models)
		if err != nil {
			log.Printf("[Seed Mongo] Error in bulk sync: %v", err)
		} else {
			log.Printf("[Seed Mongo] Successfully bulk synced %d seats to MongoDB.", len(models))
		}
	}
}
