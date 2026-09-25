package db

import (
	"log"

	"airres-api/config"

	"gorm.io/gorm"
)

// GetDBForCountry returns the appropriate Postgres Database connection and the region name
func GetDBForCountry(countryOrRegion string) (*gorm.DB, string) {
	region := countryOrRegion
	// If it's not a known region code, try to map from country name
	if region != "America" && region != "Europa" && region != "Asia" {
		region = config.GetRegionFromCountry(countryOrRegion)
	}

	if region == "America" && IsAvailable(PGAmerica) {
		log.Printf("[Router] Route request to PG America\n")
		return PGAmerica, "America"
	}

	if region == "Europa" && IsAvailable(PGEuropaAsia) {
		log.Printf("[Router] Route request to PG Europa\n")
		return PGEuropaAsia, "Europa"
	}

	if region == "Asia" {
		if IsMongoAvailable() {
			return nil, "Asia"
		}
		if IsAvailable(PGEuropaAsia) {
			return PGEuropaAsia, "Europa"
		}
	}
	if IsAvailable(PGAmerica) {
		return PGAmerica, "America"
	}
	if IsAvailable(PGEuropaAsia) {
		return PGEuropaAsia, "Europa"
	}
	return nil, region
}
