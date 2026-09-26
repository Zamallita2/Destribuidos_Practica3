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

func TestFilterAircraftOverlapsAllowsTeleportAfterLanding(t *testing.T) {
	flights := []models.Vuelo{
		{ID: 1, IDAvion: 7, IDDestino: 10, SalidaProgramada: 100, LlegadaProgramada: 200},
		{ID: 2, IDAvion: 7, IDOrigen: 20, SalidaProgramada: 200, LlegadaProgramada: 300},
		{ID: 3, IDAvion: 7, SalidaProgramada: 150, LlegadaProgramada: 250},
		{ID: 4, IDAvion: 8, SalidaProgramada: 150, LlegadaProgramada: 250},
	}
	accepted, rejected := FilterAircraftOverlaps(flights)
	if len(accepted) != 3 || len(rejected) != 1 || rejected[0].ID != 3 {
		t.Fatalf("accepted=%v rejected=%v", accepted, rejected)
	}
}
