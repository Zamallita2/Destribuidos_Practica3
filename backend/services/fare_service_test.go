package services

import "testing"

func TestRouteFareRejectsMissingClassAndDirection(t *testing.T) {
	economy := map[string]map[string]float64{"ATL": {"LAX": 400}, "LAX": {"ATL": 0}}
	first := map[string]map[string]float64{"ATL": {"LAX": 0}, "LAX": {"ATL": 900}}
	if fare, ok := RouteFare("ATL", "LAX", "REGULAR", economy, first); !ok || fare != 400 {
		t.Fatalf("economy expected 400, got %v %v", fare, ok)
	}
	if _, ok := RouteFare("ATL", "LAX", "VIP", economy, first); ok {
		t.Fatal("VIP must not be sold when its fare is null")
	}
	if _, ok := RouteFare("LAX", "ATL", "REGULAR", economy, first); ok {
		t.Fatal("reverse economy route must not be inferred")
	}
	if fare, ok := RouteFare("LAX", "ATL", "VIP", economy, first); !ok || fare != 900 {
		t.Fatalf("reverse VIP expected 900, got %v %v", fare, ok)
	}
}
