package data

import (
	"testing"

	"airres-api/models"
)

func TestFlightStatusAtBoundaries(t *testing.T) {
	for _, test := range []struct {
		departure, arrival, now int64
		want                    uint
	}{
		{100, 200, 99, 1}, {100, 200, 100, 4}, {100, 200, 199, 4}, {100, 200, 200, 5},
	} {
		if got := FlightStatusAt(test.departure, test.arrival, test.now); got != test.want {
			t.Errorf("got %d, want %d for %+v", got, test.want, test)
		}
	}
}

func testHours(from, to uint) float64 {
	if from == 0 || to == 0 || from == to {
		return 0
	}
	return 1 // one hour between any two different cities
}

func TestAssignAircraftNeverTeleports(t *testing.T) {
	flights := []models.Vuelo{
		{ID: 1, IDAvion: 7, IDOrigen: 1, IDDestino: 2, SalidaProgramada: 0, LlegadaProgramada: 3600},
		// Aircraft 7 is at city 2; city 3 is reachable with a one-hour positioning flight.
		{ID: 2, IDAvion: 7, IDOrigen: 3, IDDestino: 1, SalidaProgramada: 3 * 3600, LlegadaProgramada: 4 * 3600},
		// Not enough time to reposition aircraft 7 from city 1 to city 2.
		{ID: 3, IDAvion: 7, IDOrigen: 2, IDDestino: 1, SalidaProgramada: 4*3600 + 60, LlegadaProgramada: 5 * 3600},
	}
	accepted, ferries, rejected := AssignAircraft(flights, []uint{7}, testHours)
	if len(accepted) != 2 || len(rejected) != 1 || rejected[0].ID != 3 {
		t.Fatalf("accepted=%v rejected=%v", accepted, rejected)
	}
	if len(ferries) != 1 {
		t.Fatalf("ferries=%v", ferries)
	}
	ferry := ferries[0]
	if ferry.IDVueloSiguiente != 2 || ferry.IDAvion != 7 || ferry.IDOrigen != 2 || ferry.IDDestino != 3 ||
		ferry.SalidaProgramada != 2*3600 || ferry.LlegadaProgramada != 3*3600 {
		t.Fatalf("unexpected ferry %+v", ferry)
	}
	report := AnalyzeContinuityWithRepositioning(accepted, ferries)
	if report.Jumps != 0 || report.Overlaps != 0 || report.Repositionings != 1 {
		t.Fatalf("continuity %+v", report)
	}
}

func TestAssignAircraftUsesAnotherAircraftWaitingAtOrigin(t *testing.T) {
	flights := []models.Vuelo{
		{ID: 1, IDAvion: 7, IDOrigen: 1, IDDestino: 2, SalidaProgramada: 0, LlegadaProgramada: 3600},
		{ID: 2, IDAvion: 8, IDOrigen: 1, IDDestino: 3, SalidaProgramada: 60, LlegadaProgramada: 3600},
		// Both requested aircraft are busy or elsewhere; aircraft 8 is free at city 3.
		{ID: 3, IDAvion: 7, IDOrigen: 3, IDDestino: 1, SalidaProgramada: 3600, LlegadaProgramada: 7200},
	}
	accepted, ferries, rejected := AssignAircraft(flights, []uint{7, 8}, testHours)
	if len(accepted) != 3 || len(rejected) != 0 || len(ferries) != 0 {
		t.Fatalf("accepted=%v ferries=%v rejected=%v", accepted, ferries, rejected)
	}
	if accepted[2].ID != 3 || accepted[2].IDAvion != 8 {
		t.Fatalf("flight 3 should use aircraft 8: %+v", accepted[2])
	}
}

func TestAssignAircraftRejectsWhenNoAircraftCanArrive(t *testing.T) {
	flights := []models.Vuelo{
		{ID: 1, IDAvion: 7, IDOrigen: 1, IDDestino: 2, SalidaProgramada: 0, LlegadaProgramada: 3600},
		{ID: 2, IDAvion: 7, IDOrigen: 1, IDDestino: 2, SalidaProgramada: 1800, LlegadaProgramada: 5400},
	}
	noRoutes := func(from, to uint) float64 { return 0 }
	accepted, ferries, rejected := AssignAircraft(flights, []uint{7}, noRoutes)
	if len(accepted) != 1 || len(ferries) != 0 || len(rejected) != 1 || rejected[0].ID != 2 {
		t.Fatalf("accepted=%v ferries=%v rejected=%v", accepted, ferries, rejected)
	}
}
