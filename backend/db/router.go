package db

import (
	"airres-api/config"
	"gorm.io/gorm"
)

// ResolveReadSource keeps regional PostgreSQL primary, MongoDB as the global
// read-only fallback, and the other PostgreSQL as a final replicated fallback.
func ResolveReadSource(region string, available func(string) bool) string {
	order := []string{"pg_eu", "mongo", "pg_am"}
	if region == "America" {
		order = []string{"pg_am", "mongo", "pg_eu"}
	}
	for _, source := range order {
		if available(source) {
			return source
		}
	}
	return "none"
}

// GetDBForCountry resolves the source for region-based reads. The second
// return value is "Mongo" when the global snapshot serves the request.
func GetDBForCountry(countryOrRegion string) (*gorm.DB, string) {
	region := countryOrRegion
	// If it's not a known region code, try to map from country name
	if region != "America" && region != "Europa" && region != "Asia" {
		region = config.GetRegionFromCountry(countryOrRegion)
	}

	source := ResolveReadSource(region, func(node string) bool {
		switch node {
		case "pg_am":
			return IsAvailable(PGAmerica)
		case "pg_eu":
			return IsAvailable(PGEuropaAsia)
		case "mongo":
			return IsMongoAvailable()
		}
		return false
	})
	switch source {
	case "pg_am":
		return PGAmerica, "America"
	case "pg_eu":
		return PGEuropaAsia, "Europa"
	case "mongo":
		return nil, "Mongo"
	}
	return nil, region
}
