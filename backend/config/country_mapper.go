package config

import "strings"

// GetRegionFromCountry maps a country to a region
func GetRegionFromCountry(country string) string {
	c := strings.ToLower(strings.TrimSpace(country))

	// America
	americanCountries := []string{"estados unidos", "usa", "us", "brasil", "brazil", "mexico", "méxico", "mxico", "colombia", "argentina", "canada"}
	for _, a := range americanCountries {
		if c == a {
			return "America"
		}
	}

	// Asia
	asiaCountries := []string{"china", "japón", "japon", "corea", "india", "tailandia", "singapur", "vietnam", "emiratos", "arabia", "australia"}
	for _, a := range asiaCountries {
		if strings.Contains(c, a) {
			return "Asia"
		}
	}

	// Default to Europa for the other PG Node
	return "Europa"
}
