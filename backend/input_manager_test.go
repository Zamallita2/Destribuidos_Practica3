package main

import (
	"testing"

	"airres-api/data"
)

func TestPreviewDatasetRejectsOverlappingAircraftAcrossOrigins(t *testing.T) {
	hours := 2.0
	fare := 50.0
	matrix := data.MatricesJSON{
		TravelTime:   map[string]map[string]*float64{"ATL": {"DFW": &hours}, "TYO": {"DFW": &hours}},
		EconomyFares: map[string]map[string]*float64{"ATL": {"DFW": &fare}, "TYO": {"DFW": &fare}},
	}
	csv := "flight_date,flight_time,origin,destination,aircraft_id,status,gate\n09/12/26,00:00,ATL,DFW,7,SCHEDULED,G1\n09/12/26,01:00,TYO,DFW,7,SCHEDULED,G1\n09/12/26,02:00,TYO,DFW,7,SCHEDULED,G1\n"
	rows, eligible, rejected, err := previewDataset([]byte(csv), matrix, map[uint]bool{7: true})
	if err != nil || rows != 3 || eligible != 2 || rejected != 1 {
		t.Fatalf("rows=%d eligible=%d rejected=%d err=%v", rows, eligible, rejected, err)
	}
}
