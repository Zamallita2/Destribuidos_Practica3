package data

import "testing"

func TestRouteAvailabilityAllowsEitherClassAndKeepsDirection(t *testing.T) {
	eco := 400.0
	first := 900.0
	m := MatricesJSON{
		EconomyFares: map[string]map[string]*float64{
			"ATL": {"LAX": &eco, "PAR": nil},
			"LAX": {"ATL": nil},
		},
		FirstClassFares: map[string]map[string]*float64{
			"ATL": {"LAX": nil, "PAR": &first},
			"LAX": {"ATL": nil},
		},
		TravelTime: map[string]map[string]float64{
			"ATL": {"LAX": 5, "PAR": 9},
			"LAX": {"ATL": 5},
		},
	}
	tests := []struct {
		from, to              string
		allowed, economy, vip bool
		hours                 float64
	}{
		{"ATL", "LAX", true, true, false, 5},
		{"ATL", "PAR", true, false, true, 9},
		{"LAX", "ATL", false, false, false, 0},
	}
	for _, test := range tests {
		allowed, economy, vip, hours := m.RouteAvailability(test.from, test.to)
		if allowed != test.allowed || economy != test.economy || vip != test.vip || hours != test.hours {
			t.Errorf("%s -> %s: got %v %v %v %v", test.from, test.to, allowed, economy, vip, hours)
		}
	}
}
