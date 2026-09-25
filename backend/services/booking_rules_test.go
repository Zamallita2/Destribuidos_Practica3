package services

import (
	"testing"
	"time"

	"airres-api/models"
)

func TestBookingRulesRequireFutureFlightAndMatchingSeat(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	flight := models.Vuelo{IDAvion: 7, IDEstadoVuelo: 1, SalidaProgramada: now.Add(time.Hour).Unix()}
	seat := models.Asiento{IDAvion: 7, Clase: "REGULAR", Estado: "AVAILABLE"}
	if err := ValidateBooking(flight, seat, "RESERVED", now); err != nil {
		t.Fatalf("valid booking rejected: %v", err)
	}
	seat.IDAvion = 8
	if err := ValidateBooking(flight, seat, "RESERVED", now); err == nil {
		t.Fatal("seat from another aircraft accepted")
	}
	seat.IDAvion = 7
	seat.Estado = "SALED"
	if err := ValidateBooking(flight, seat, "RESERVED", now); err == nil {
		t.Fatal("seeded sold seat accepted")
	}
	seat.Estado = "AVAILABLE"
	flight.SalidaProgramada = now.Add(-time.Hour).Unix()
	if err := ValidateBooking(flight, seat, "SALED", now); err == nil {
		t.Fatal("past flight accepted")
	}
	flight.SalidaProgramada = now.Add(time.Hour).Unix()
	flight.IDEstadoVuelo = 2
	if err := ValidateBooking(flight, seat, "SALED", now); err == nil {
		t.Fatal("boarding flight accepted")
	}
}
