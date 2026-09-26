package main

import (
	"testing"

	"airres-api/data"
)

// Aircraft 7 lands in DFW and there is no DFW->TYO route to reposition it.
func TestPreviewDatasetRejectsAircraftThatCannotReachOrigin(t *testing.T) {
	hours := 2.0
	fare := 50.0
	matrix := data.MatricesJSON{
		TravelTime:   map[string]map[string]*float64{"ATL": {"DFW": &hours}, "TYO": {"DFW": &hours}},
		EconomyFares: map[string]map[string]*float64{"ATL": {"DFW": &fare}, "TYO": {"DFW": &fare}},
	}
	csv := "flight_date,flight_time,origin,destination,aircraft_id,status,gate\n09/12/26,00:00,ATL,DFW,7,SCHEDULED,G1\n09/12/26,01:00,TYO,DFW,7,SCHEDULED,G1\n09/12/26,02:00,TYO,DFW,7,SCHEDULED,G1\n"
	rows, eligible, rejected, err := previewDataset([]byte(csv), matrix, map[uint]bool{7: true})
	if err != nil || rows != 3 || eligible != 1 || rejected != 2 {
		t.Fatalf("rows=%d eligible=%d rejected=%d err=%v", rows, eligible, rejected, err)
	}
}

// The busy 17:30 row takes free aircraft 18, so the other 17:30 row and the
// 19:00 row have no aircraft in ATL (there is no DFW->ATL route back).
func TestPreviewDatasetReassignsFreeAircraftAtOrigin(t *testing.T) {
	hours, fare := 2.0, 50.0
	matrix := data.MatricesJSON{TravelTime: map[string]map[string]*float64{"ATL": {"DFW": &hours}}, EconomyFares: map[string]map[string]*float64{"ATL": {"DFW": &fare}}}
	csv := "flight_date,flight_time,origin,destination,aircraft_id,status,gate\n10/06/26,17:00,ATL,DFW,7,SCHEDULED,G1\n10/06/26,17:30,ATL,DFW,7,SCHEDULED,G1\n10/06/26,17:30,ATL,DFW,18,SCHEDULED,G1\n10/06/26,19:00,ATL,DFW,7,SCHEDULED,G1\n"
	rows, eligible, rejected, err := previewDataset([]byte(csv), matrix, map[uint]bool{7: true, 18: true})
	if err != nil || rows != 4 || eligible != 2 || rejected != 2 {
		t.Fatalf("rows=%d eligible=%d rejected=%d err=%v", rows, eligible, rejected, err)
	}
}

func TestPreviewDatasetRepositionsAircraftWhenRouteExists(t *testing.T) {
	hours, fare := 2.0, 50.0
	matrix := data.MatricesJSON{
		TravelTime:   map[string]map[string]*float64{"ATL": {"DFW": &hours}, "DFW": {"TYO": &hours}, "TYO": {"DFW": &hours}},
		EconomyFares: map[string]map[string]*float64{"ATL": {"DFW": &fare}, "TYO": {"DFW": &fare}},
	}
	// DFW->TYO has no fare, but an empty positioning flight only needs the travel time.
	csv := "flight_date,flight_time,origin,destination,aircraft_id,status,gate\n09/12/26,00:00,ATL,DFW,7,SCHEDULED,G1\n09/12/26,04:00,TYO,DFW,7,SCHEDULED,G1\n"
	rows, eligible, rejected, err := previewDataset([]byte(csv), matrix, map[uint]bool{7: true})
	if err != nil || rows != 2 || eligible != 2 || rejected != 0 {
		t.Fatalf("rows=%d eligible=%d rejected=%d err=%v", rows, eligible, rejected, err)
	}
}
