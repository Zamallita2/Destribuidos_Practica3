package main

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"time"

	"airres-api/data"
	"airres-api/db"
	"airres-api/models"
	"gorm.io/gorm"
)

type airportMetadata struct {
	Country  string `json:"country"`
	Region   string `json:"region"`
	TimeZone string `json:"time_zone"`
}

func seedAirportCatalog() {
	matrixBytes, matrixErr := os.ReadFile(filepath.Join("data", "matrices.json"))
	metadataBytes, metadataErr := os.ReadFile(filepath.Join("data", "airports.json"))
	if matrixErr != nil || metadataErr != nil {
		log.Printf("[Airports] metadata unavailable: %v %v", matrixErr, metadataErr)
		return
	}
	var matrix data.MatricesJSON
	var metadata map[string]airportMetadata
	if err := json.Unmarshal(matrixBytes, &matrix); err != nil {
		log.Printf("[Airports] matrix invalid: %v", err)
		return
	}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		log.Printf("[Airports] catalog invalid: %v", err)
		return
	}
	for _, conn := range []*gorm.DB{db.PGAmerica, db.PGEuropaAsia} {
		if !db.IsAvailable(conn) {
			continue
		}
		for _, code := range matrix.Airports {
			meta, ok := metadata[code]
			if !ok || meta.Region == "" || meta.Country == "" || meta.TimeZone == "" {
				log.Printf("[Airports] %s missing country, region or IANA time zone", code)
				continue
			}
			if _, err := time.LoadLocation(meta.TimeZone); err != nil {
				log.Printf("[Airports] %s invalid time zone: %v", code, err)
				continue
			}
			var city models.Ciudad
			if err := conn.Where("codigo = ?", code).First(&city).Error; err == gorm.ErrRecordNotFound {
				hash := fnv.New32a()
				_, _ = hash.Write([]byte(code))
				city = models.Ciudad{ID: uint(50000 + hash.Sum32()%1000000), Codigo: code}
			} else if err != nil {
				log.Printf("[Airports] read %s failed: %v", code, err)
				continue
			}
			city.Pais, city.Region, city.TimeZone = meta.Country, meta.Region, meta.TimeZone
			if err := conn.Save(&city).Error; err != nil {
				log.Printf("[Airports] save %s failed: %v", code, err)
				continue
			}
			var count int64
			conn.Model(&models.Puerta{}).Where("id_ciudad = ?", city.ID).Count(&count)
			if count == 0 {
				conn.Create(&models.Puerta{Puerta: "G1", IDCiudad: city.ID})
			}
		}
	}
}
