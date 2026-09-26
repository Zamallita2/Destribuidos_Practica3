package data

import (
	"sort"

	"airres-api/models"
)

// FlightStatusAt assigns imported flights from their scheduled interval.
func FlightStatusAt(departure, arrival, now int64) uint {
	if arrival <= now {
		return 5
	}
	if departure <= now {
		return 4
	}
	return 1
}

func FlightsOverlap(a, b models.Vuelo) bool {
	return a.IDAvion == b.IDAvion && a.SalidaProgramada < b.LlegadaProgramada && b.SalidaProgramada < a.LlegadaProgramada
}

// FilterAircraftOverlaps processes valid rows in CSV order. Only accepted
// intervals of the same aircraft are compared; locations are deliberately ignored.
func FilterAircraftOverlaps(flights []models.Vuelo) ([]models.Vuelo, []models.Vuelo) {
	accepted := make([]models.Vuelo, 0, len(flights))
	rejected := []models.Vuelo{}
	byAircraft := make(map[uint][]models.Vuelo)
	for _, flight := range flights {
		flown := byAircraft[flight.IDAvion]
		position := sort.Search(len(flown), func(i int) bool { return flown[i].SalidaProgramada >= flight.SalidaProgramada })
		if position > 0 && FlightsOverlap(flown[position-1], flight) || position < len(flown) && FlightsOverlap(flown[position], flight) {
			rejected = append(rejected, flight)
			continue
		}
		accepted = append(accepted, flight)
		flown = append(flown, models.Vuelo{})
		copy(flown[position+1:], flown[position:])
		flown[position] = flight
		byAircraft[flight.IDAvion] = flown
	}
	return accepted, rejected
}
