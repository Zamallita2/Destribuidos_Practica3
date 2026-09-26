package services

import (
	"reflect"
	"strings"
	"testing"

	"airres-api/models"
)

func TestFlightManifestHasRoundedTargetsAndUnicodePassengers(t *testing.T) {
	seats := make([]models.Asiento, 228)
	for i := range seats {
		seats[i].ID = uint(i + 1)
		seats[i].Clase = "REGULAR"
	}
	manifest := BuildFlightManifest(42, seats, nil)
	if manifest.EligibleSeats != 228 || manifest.SoldCount != 166 || manifest.ReservedCount != 7 {
		t.Fatalf("unexpected counts: %+v", manifest)
	}
	entries, err := DecodeManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 173 {
		t.Fatalf("got %d occupied seats", len(entries))
	}
	seen := map[uint]bool{}
	unicode := false
	for _, entry := range entries {
		if seen[entry.SeatID] {
			t.Fatalf("duplicate seat %d", entry.SeatID)
		}
		seen[entry.SeatID] = true
		if strings.ContainsAny(entry.PassengerName, "张أ佐áü") {
			unicode = true
		}
	}
	if !unicode {
		t.Fatal("expected diverse Unicode passenger names")
	}
	again := BuildFlightManifest(42, seats, nil)
	if !reflect.DeepEqual(again.Assignments, manifest.Assignments) {
		t.Fatal("seed must be deterministic")
	}
}

func TestOccupancyTargetsUseConfiguredPercentages(t *testing.T) {
	t.Setenv("PERCENTAGE_SOLD", "50")
	t.Setenv("PERCENTAGE_RESERVED", "10")
	if sold, reserved := OccupancyTargets(200); sold != 100 || reserved != 20 {
		t.Fatalf("got sold=%d reserved=%d", sold, reserved)
	}
	t.Setenv("PERCENTAGE_SOLD", "invalid")
	t.Setenv("PERCENTAGE_RESERVED", "")
	if sold, reserved := OccupancyTargets(228); sold != 166 || reserved != 7 {
		t.Fatalf("defaults: got sold=%d reserved=%d", sold, reserved)
	}
	t.Setenv("PERCENTAGE_SOLD", "99")
	t.Setenv("PERCENTAGE_RESERVED", "5")
	if sold, reserved := OccupancyTargets(228); sold != 166 || reserved != 7 {
		t.Fatalf("over 100%% must fall back: got sold=%d reserved=%d", sold, reserved)
	}
}
