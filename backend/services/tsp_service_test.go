package services

import "testing"

func TestTSPTimeDoesNotUseRouteWithoutFareInSelectedClass(t *testing.T) {
	times := map[string]map[string]float64{
		"A": {"B": 1, "C": 5},
		"B": {"A": 1, "C": 1},
		"C": {"A": 5, "B": 1},
	}
	economy := map[string]map[string]float64{
		"A": {"B": 0, "C": 10},
		"B": {"A": 10, "C": 10},
		"C": {"A": 10, "B": 10},
	}
	route := CalculateTSPFromMatrices([]string{"A", "B", "C"}, "TIME", "REGULAR", false, times, economy, nil)
	if route == nil {
		t.Fatal("expected a feasible open path")
	}
	for _, leg := range route.Vuelos {
		if leg.Salida == "A" && leg.Llegada == "B" {
			t.Fatal("used A -> B although its economy fare is null")
		}
	}
}

func TestTSPCanRequireDirectedReturn(t *testing.T) {
	times := map[string]map[string]float64{
		"A": {"B": 1},
		"B": {"C": 1},
		"C": {},
	}
	fares := map[string]map[string]float64{
		"A": {"B": 10},
		"B": {"C": 10},
		"C": {},
	}
	if got := CalculateTSPFromMatrices([]string{"A", "B", "C"}, "COST", "REGULAR", true, times, fares, nil); got != nil {
		t.Fatalf("closed tour requires a return arc, got %+v", got)
	}
	if got := CalculateTSPFromMatrices([]string{"A", "B", "C"}, "COST", "REGULAR", false, times, fares, nil); got == nil {
		t.Fatal("expected open path")
	}
}
