package data

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRejectedCSVOnlyCityAndMatrix(t *testing.T) {
	dir := t.TempDir()
	matrix := `{"airports":["AAA","BBB"],"travel_time":{"AAA":{"BBB":2},"BBB":{"AAA":0}},"economy_fares":{"AAA":{"BBB":null},"BBB":{"AAA":80}},"first_class_fares":{"AAA":{"BBB":100},"BBB":{"AAA":null}}}`
	if err := os.WriteFile(filepath.Join(dir, "matrices.json"), []byte(matrix), 0644); err != nil {
		t.Fatal(err)
	}
	rows := "date,time,origin,destination,aircraft,status,gate\n1,2,AAA,BBB,1,SCHEDULED,G1\n1,2,AAA,CCC,1,SCHEDULED,G1\n1,2,AAA,AAA,1,SCHEDULED,G1\n1,2,BBB,AAA,1,SCHEDULED,G1\n"
	if err := os.WriteFile(filepath.Join(dir, "flights.csv"), []byte(rows), 0644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "reports", "rejected.csv")
	count, err := GenerateRejectedCSV(filepath.Join(dir, "flights.csv"), filepath.Join(dir, "matrices.json"), output)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("rejected %d rows, want 3", count)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || !strings.Contains(got[1][7], "CCC") || got[2][7] == "" || !strings.Contains(got[3][7], "duración") {
		t.Fatalf("unexpected report: %#v", got)
	}
}
