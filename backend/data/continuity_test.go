package data

import (
	"testing"

	"airres-api/models"
)

func TestAnalyzeContinuityReportsButKeepsFlights(t *testing.T) {
	flights := []models.Vuelo{
		{ID: 2, IDAvion: 1, IDOrigen: 3, IDDestino: 4, SalidaProgramada: 20, LlegadaProgramada: 30},
		{ID: 1, IDAvion: 1, IDOrigen: 1, IDDestino: 2, SalidaProgramada: 10, LlegadaProgramada: 25},
		{ID: 3, IDAvion: 2, IDOrigen: 1, IDDestino: 2, SalidaProgramada: 15, LlegadaProgramada: 20},
	}
	report := AnalyzeContinuity(flights)
	if report.Flights != 3 || report.Jumps != 1 || report.Overlaps != 1 || len(report.Examples) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(flights) != 3 {
		t.Fatal("diagnostic must not discard flights")
	}
}
