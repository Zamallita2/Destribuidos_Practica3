package db

import (
	"reflect"
	"testing"
)

func TestPurchaseWriteOrderUsesBuyerLocationBeforeFlightOrigin(t *testing.T) {
	tests := []struct {
		name, zone string
		flightID   uint
		want       []string
	}{
		{"Spain buying American flight", "Europe/Madrid", 42, []string{"pg_eu", "pg_am"}},
		{"Bolivia buying Asian flight", "America/La_Paz", 1000000042, []string{"pg_am", "pg_eu"}},
		{"Japan buying American flight", "Asia/Tokyo", 42, []string{"pg_eu", "pg_am"}},
		{"South Africa buying American flight", "Africa/Johannesburg", 42, []string{"pg_eu", "pg_am"}},
		{"no buyer location preserves owner routing", "UTC", 1000000042, []string{"pg_eu", "pg_am"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PurchaseWriteOrder(test.zone, test.flightID); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
