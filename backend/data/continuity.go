package data

import (
	"sort"

	"airres-api/models"
)

type ContinuityIssue struct {
	AircraftID          uint `json:"aircraft_id"`
	PreviousFlightID    uint `json:"previous_flight_id"`
	NextFlightID        uint `json:"next_flight_id"`
	PreviousDestination uint `json:"previous_destination"`
	NextOrigin          uint `json:"next_origin"`
	LocationMismatch    bool `json:"location_mismatch"`
	TimeOverlap         bool `json:"time_overlap"`
}

type ContinuityReport struct {
	Flights     int `json:"flights"`
	Transitions int `json:"transitions"`
	Jumps       int `json:"jumps"`
	Overlaps    int `json:"overlaps"`
	// Repositionings counts empty positioning flights included in the analysis.
	Repositionings int               `json:"repositionings"`
	Examples       []ContinuityIssue `json:"examples"`
}

// AnalyzeContinuity is diagnostic only. Cancelled flights do not move an
// aircraft, and the input slice is never changed or filtered.
func AnalyzeContinuity(flights []models.Vuelo) ContinuityReport {
	return AnalyzeContinuityWithRepositioning(flights, nil)
}

// AnalyzeContinuityWithRepositioning also treats empty positioning flights as
// aircraft movements, so a repositioned aircraft is not reported as a jump.
func AnalyzeContinuityWithRepositioning(flights []models.Vuelo, ferries []models.Reposicionamiento) ContinuityReport {
	ordered := append([]models.Vuelo(nil), flights...)
	for _, ferry := range ferries {
		ordered = append(ordered, models.Vuelo{
			ID: ferry.IDVueloSiguiente, IDAvion: ferry.IDAvion, IDOrigen: ferry.IDOrigen, IDDestino: ferry.IDDestino,
			SalidaProgramada: ferry.SalidaProgramada, LlegadaProgramada: ferry.LlegadaProgramada,
		})
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].IDAvion != ordered[j].IDAvion {
			return ordered[i].IDAvion < ordered[j].IDAvion
		}
		if ordered[i].SalidaProgramada != ordered[j].SalidaProgramada {
			return ordered[i].SalidaProgramada < ordered[j].SalidaProgramada
		}
		return ordered[i].ID < ordered[j].ID
	})
	report := ContinuityReport{Flights: len(flights), Repositionings: len(ferries), Examples: []ContinuityIssue{}}
	var previous models.Vuelo
	for _, current := range ordered {
		if current.IDEstadoVuelo == 7 {
			continue
		}
		if previous.IDAvion == current.IDAvion && previous.ID != 0 {
			report.Transitions++
			issue := ContinuityIssue{
				AircraftID:       current.IDAvion,
				PreviousFlightID: previous.ID, NextFlightID: current.ID,
				PreviousDestination: previous.IDDestino, NextOrigin: current.IDOrigen,
				LocationMismatch: previous.IDDestino != current.IDOrigen,
				TimeOverlap:      previous.LlegadaProgramada > current.SalidaProgramada,
			}
			if issue.LocationMismatch {
				report.Jumps++
			}
			if issue.TimeOverlap {
				report.Overlaps++
			}
			if (issue.LocationMismatch || issue.TimeOverlap) && len(report.Examples) < 100 {
				report.Examples = append(report.Examples, issue)
			}
		}
		previous = current
	}
	return report
}
