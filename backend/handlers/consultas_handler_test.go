package handlers

import (
	"testing"
	"time"
)

func TestAggregateClassesCountsEachClassSeparately(t *testing.T) {
	rows := aggregateClasses([]reportRow{
		{"clase": "VIP", "vendidos": int64(3), "ingresos": "120.50"},
		{"clase": "VIP", "vendidos": int64(2), "ingresos": "80.25"},
		{"clase": "REGULAR", "vendidos": int64(4), "ingresos": "60.00"},
	})
	if len(rows) != 2 || rows[0]["clase"] != "Ejecutiva" || numeric(rows[0], "asientos_vendidos") != 5 || numeric(rows[0], "ingresos") != 200.75 {
		t.Fatalf("incorrect executive totals: %#v", rows)
	}
	if rows[1]["clase"] != "Turística" || numeric(rows[1], "asientos_vendidos") != 4 {
		t.Fatalf("incorrect economy totals: %#v", rows)
	}
}

func TestAggregateQuartersUsesUTCFlightDate(t *testing.T) {
	q1 := time.Date(2025, 3, 31, 23, 59, 0, 0, time.UTC).Unix()
	q2 := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC).Unix()
	rows := aggregateQuarters([]reportRow{
		{"salida": q1, "ingresos": "10.25"},
		{"salida": q2, "ingresos": "20.50"},
	})
	if len(rows) != 2 || rows[0]["trimestre"] != "Q1 2025" || rows[1]["trimestre"] != "Q2 2025" {
		t.Fatalf("incorrect quarter boundaries: %#v", rows)
	}
}

func TestOneRowPerFlightKeepsDifferentFlights(t *testing.T) {
	rows := oneRowPerFlight([]reportRow{
		{"vuelo": int64(11), "asiento": "2A"},
		{"vuelo": int64(11), "asiento": "3B"},
		{"vuelo": int64(12), "asiento": "4C"},
	})
	if len(rows) != 2 || rows[0]["asiento"] != "2A" || rows[1]["vuelo"] != int64(12) {
		t.Fatalf("expected one representative seat per flight: %#v", rows)
	}
}
