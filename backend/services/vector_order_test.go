package services

import "testing"

func TestCompareEventVersionsDetectsCausalityAndBreaksConcurrentTie(t *testing.T) {
	if !ShouldApplyVersion(`{"a":2}`, `{"a":1}`, 2, 1, "a", "a") {
		t.Fatal("causally newer event rejected")
	}
	if ShouldApplyVersion(`{"a":1}`, `{"a":2}`, 20, 1, "a", "a") {
		t.Fatal("causally older event accepted because Lamport was larger")
	}
	if !ShouldApplyVersion(`{"b":1}`, `{"a":1}`, 7, 7, "b", "a") {
		t.Fatal("concurrent tie should be deterministic by node id")
	}
	if ShouldApplyVersion(`{"a":1}`, `{"b":1}`, 7, 7, "a", "b") {
		t.Fatal("concurrent tie loser accepted")
	}
}
