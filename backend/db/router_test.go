package db

import "testing"

func TestResolveReadSource(t *testing.T) {
	tests := []struct {
		name, region, want string
		up                 map[string]bool
	}{
		{"america primary", "America", "pg_am", map[string]bool{"pg_am": true, "pg_eu": true, "mongo": true}},
		{"asia primary", "Asia", "pg_eu", map[string]bool{"pg_am": true, "pg_eu": true, "mongo": true}},
		{"europe primary", "Europa", "pg_eu", map[string]bool{"pg_am": true, "pg_eu": true, "mongo": true}},
		{"america mongo fallback", "America", "mongo", map[string]bool{"pg_eu": true, "mongo": true}},
		{"asia mongo fallback", "Asia", "mongo", map[string]bool{"pg_am": true, "mongo": true}},
		{"america last postgres", "America", "pg_eu", map[string]bool{"pg_eu": true}},
		{"asia last postgres", "Asia", "pg_am", map[string]bool{"pg_am": true}},
		{"all down", "Asia", "none", map[string]bool{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolveReadSource(test.region, func(node string) bool { return test.up[node] })
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFlightLockOrderUsesOwnerRegionFirst(t *testing.T) {
	if order := FlightLockOrder(100000005); order[0] != "pg_am" || order[1] != "pg_eu" {
		t.Fatalf("american flight: %v", order)
	}
	if order := FlightLockOrder(1100000005); order[0] != "pg_eu" || order[1] != "pg_am" {
		t.Fatalf("european/asian flight: %v", order)
	}
}
