package services

import (
	"testing"

	"airres-api/models"
)

func TestOutboxEventRoundTripPreservesTicketVersion(t *testing.T) {
	ticket := &models.Boleto{IDBoleto: 42, Estado: "REFUNDED", AvailableAt: 1700,
		LamportClock: 9, VectorClock: `{"america":3}`, SourceNode: "america"}
	row, err := EncodeOutboxEvent("UPDATE", "Boleto", ticket)
	if err != nil {
		t.Fatal(err)
	}
	event, err := DecodeOutboxEvent(row)
	if err != nil {
		t.Fatal(err)
	}
	decoded, ok := event.Data.(*models.Boleto)
	if !ok || decoded.IDBoleto != 42 || decoded.Estado != "REFUNDED" ||
		decoded.AvailableAt != 1700 || event.NodeID != "america" || event.LamportClock != 9 {
		t.Fatalf("round trip lost data: %+v", event)
	}
}
