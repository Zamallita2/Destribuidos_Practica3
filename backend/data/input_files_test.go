package data

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestCurrentMatrixIsValid(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("matrices.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MatrixFromJSON(content); err != nil {
		t.Fatal(err)
	}
}

func TestDatasetCSVReadsCSVAndExcel(t *testing.T) {
	csvInput := "flight_date, flight_time, origin, destination, aircraft_id, status, gate\n03/30/26,17:33,ATL,DFW,7,DELAYED,G26\n"
	canonical, count, err := DatasetCSV([]byte(csvInput), "flights.csv")
	if err != nil || count != 1 || !strings.Contains(string(canonical), "ATL,DFW,7") {
		t.Fatalf("csv: count=%d err=%v content=%s", count, err, canonical)
	}
	f := excelize.NewFile()
	for col, value := range FlightColumns {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := f.SetCellValue("Sheet1", cell, value); err != nil {
			t.Fatal(err)
		}
	}
	for col, value := range []string{"03/30/26", "17:33", "ATL", "DFW", "7", "DELAYED", "G26"} {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		if err := f.SetCellValue("Sheet1", cell, value); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	converted, count, err := DatasetCSV(buf.Bytes(), "flights.xlsx")
	if err != nil || count != 1 || string(converted) != string(canonical) {
		t.Fatalf("xlsx: count=%d err=%v content=%s", count, err, converted)
	}
}

func TestDatasetCSVNormalizesSpanishColumns(t *testing.T) {
	input := "Puerta,Origen,Fecha,ID_Avion,Destino,Estado,Hora\nG1,atl,03/30/26,7,dfw,scheduled,17:33\n"
	content, count, err := DatasetCSV([]byte(input), "vuelos.csv")
	if err != nil || count != 1 || !strings.Contains(string(content), "03/30/26,17:33,ATL,DFW,7,SCHEDULED,G1") {
		t.Fatalf("count=%d err=%v content=%s", count, err, content)
	}
}

func TestMatrixGridAndValidationRejectMissingCell(t *testing.T) {
	grid, err := MatrixGrid([]byte(",ATL,DFW\nATL,0,5\nDFW,5,0\n"), "times.csv", "travel_time")
	if err != nil {
		t.Fatal(err)
	}
	times := grid
	if times["ATL"]["DFW"] == nil || *times["ATL"]["DFW"] != 5 {
		t.Fatalf("unexpected time: %v", times)
	}
	eco := 100.0
	m := MatricesJSON{Airports: []string{"ATL", "DFW"}, TravelTime: times,
		EconomyFares:    map[string]map[string]*float64{"ATL": {"ATL": nil, "DFW": &eco}, "DFW": {"ATL": &eco, "DFW": nil}},
		FirstClassFares: map[string]map[string]*float64{"ATL": {"ATL": nil, "DFW": nil}, "DFW": {"ATL": nil, "DFW": nil}}}
	if err := ValidateMatrices(m); err != nil {
		t.Fatal(err)
	}
	delete(m.EconomyFares["ATL"], "DFW")
	if err := ValidateMatrices(m); err == nil {
		t.Fatal("missing fare cell was accepted")
	}
}

func TestMatrixExcelAllowsBlankTrailingFare(t *testing.T) {
	f := excelize.NewFile()
	for cell, value := range map[string]string{"B1": "ATL", "C1": "DFW", "A2": "ATL", "B2": "0", "A3": "DFW", "B3": "100"} {
		if err := f.SetCellValue("Sheet1", cell, value); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	grid, err := MatrixGrid(buf.Bytes(), "fare.xlsx", "economy_fares")
	if err != nil {
		t.Fatal(err)
	}
	if grid["ATL"]["DFW"] != nil || grid["DFW"]["DFW"] != nil {
		t.Fatalf("trailing blanks not preserved: %v", grid)
	}
}

func TestMatrixGridInvalidCellsBecomeNullAndFarePrecisionIsLimited(t *testing.T) {
	grid, err := MatrixGrid([]byte(",ATL,DFW,LAX,TYO\nATL,12.34,12.345,unknown,12.340\n"), "fare.csv", "economy_fares")
	if err != nil {
		t.Fatal(err)
	}
	if grid["ATL"]["ATL"] == nil || *grid["ATL"]["ATL"] != 12.34 || grid["ATL"]["DFW"] != nil || grid["ATL"]["LAX"] != nil || grid["ATL"]["TYO"] != nil {
		t.Fatalf("unexpected fares: %v", grid)
	}
	jsonGrid, err := MatrixGrid([]byte(`{"ATL":{"DFW":12.345,"LAX":"bad"}}`), "fare.json", "economy_fares")
	if err != nil || jsonGrid["ATL"]["DFW"] != nil || jsonGrid["ATL"]["LAX"] != nil {
		t.Fatalf("unexpected JSON fares: %v, %v", jsonGrid, err)
	}
	combined, err := MatrixFromJSON([]byte(`{"airports":["ATL","DFW"],"travel_time":{"ATL":{"ATL":null,"DFW":"bad"},"DFW":{"ATL":2,"DFW":null}},"economy_fares":{"ATL":{"ATL":null,"DFW":12.345},"DFW":{"ATL":20,"DFW":null}},"first_class_fares":{"ATL":{"ATL":null,"DFW":50},"DFW":{"ATL":null,"DFW":null}}}`))
	if err != nil || combined.TravelTime["ATL"]["DFW"] != nil || combined.EconomyFares["ATL"]["DFW"] != nil {
		t.Fatalf("unexpected combined matrix: %v, %v", combined, err)
	}
}
