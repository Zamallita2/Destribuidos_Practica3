package services

import (
	"testing"
	"time"
)

func TestRefundAvailabilityUsesConfiguredDelay(t *testing.T) {
	now := time.Unix(1000, 0)
	if got := RefundAvailableAt(now, 15); got != now.Add(15*time.Minute).Unix() {
		t.Fatalf("got %d", got)
	}
	if got := RefundAvailableAt(now, 0); got != now.Unix() {
		t.Fatalf("zero delay should release immediately, got %d", got)
	}
}
