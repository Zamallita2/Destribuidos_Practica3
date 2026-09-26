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

// FilterAircraftOverlaps keeps the first CSV row for each occupied interval.
// Aircraft locations are deliberately ignored.
func FilterAircraftOverlaps(flights []models.Vuelo) ([]models.Vuelo, []models.Vuelo) {
	ordered := append([]models.Vuelo(nil), flights...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].IDAvion != ordered[j].IDAvion {
			return ordered[i].IDAvion < ordered[j].IDAvion
		}
		return ordered[i].SalidaProgramada < ordered[j].SalidaProgramada
	})
	accepted := make([]models.Vuelo, 0, len(ordered))
	rejected := []models.Vuelo{}
	lastArrival := map[uint]int64{}
	for _, flight := range ordered {
		if last, exists := lastArrival[flight.IDAvion]; exists && flight.SalidaProgramada < last {
			rejected = append(rejected, flight)
			continue
		}
		accepted = append(accepted, flight)
		lastArrival[flight.IDAvion] = flight.LlegadaProgramada
	}
	return accepted, rejected
}
