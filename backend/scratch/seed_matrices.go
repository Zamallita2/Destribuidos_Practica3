package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Matrices struct {
	TravelTime     interface{} `json:"travel_time"`
	PreciosRegular interface{} `json:"precios_regular"`
	PreciosVip     interface{} `json:"precios_vip"`
}

func main() {
	// 1. Read matrices.json
	data, err := os.ReadFile("../data/matrices.json")
	if err != nil {
		log.Fatalf("Error reading matrices.json: %v", err)
	}

	var m RawMatrices
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatalf("Error unmarshaling matrices: %v", err)
	}

	timeJson, _ := json.Marshal(m.TravelTime)
	precRegJson, _ := json.Marshal(m.PreciosRegular)
	precVipJson, _ := json.Marshal(m.PreciosVip)

	// 2. Connect to Postgres
	conns := []string{
		"host=localhost user=postgres password=root dbname=airres_am port=5432 sslmode=disable",
		"host=localhost user=postgres password=root dbname=airres_eu port=5433 sslmode=disable",
	}

	for _, dsn := range conns {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Printf("Error connecting to PG (%s): %v", dsn, err)
			continue
		}

		db.Exec("TRUNCATE TABLE precios, detalles_vuelos CASCADE")
		db.Exec("INSERT INTO precios (matriz_precios_regular, matriz_precios_vip) VALUES (?, ?)", string(precRegJson), string(precVipJson))
		db.Exec("INSERT INTO detalles_vuelos (matriz_tiempos) VALUES (?)", string(timeJson))
		fmt.Printf("Seeded PG: %s\n", dsn)
	}

	// 3. Connect to Mongo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:root@localhost:27017"))
	if err == nil {
		db := client.Database("airres_sync")
		db.Collection("precios").Drop(ctx)
		db.Collection("detalles_vuelos").Drop(ctx)

		db.Collection("precios").InsertOne(ctx, map[string]interface{}{
			"matriz_precios_regular": m.PreciosRegular,
			"matriz_precios_vip":     m.PreciosVip,
		})
		db.Collection("detalles_vuelos").InsertOne(ctx, map[string]interface{}{
			"matriz_tiempos": m.TravelTime,
		})
		fmt.Println("Seeded MongoDB Asia")
	} else {
		log.Printf("Error connecting to Mongo: %v", err)
	}
}

type RawMatrices struct {
	TravelTime     map[string]interface{} `json:"travel_time"`
	PreciosRegular map[string]interface{} `json:"precios_regular"`
	PreciosVip     map[string]interface{} `json:"precios_vip"`
}
