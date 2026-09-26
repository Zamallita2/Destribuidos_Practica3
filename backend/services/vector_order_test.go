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

func TestObserveClockMergesThreeNodes(t *testing.T) {
	vcMutex.Lock()
	previousNode, previousClock := NodeID, GlobalVectorClock
	NodeID, GlobalVectorClock = "europa", VectorClock{"europa": 2}
	vcMutex.Unlock()
	defer func() { NodeID, GlobalVectorClock = previousNode, previousClock }()

	// Europa reads a ticket last written by America and Asia, then updates it.
	ObserveClock(10, `{"america":4,"asia":1}`)
	next := TickVectorClock()
	if !IsVectorDominant(next, `{"america":4,"asia":1}`) {
		t.Fatalf("the update must causally follow what it read: %s", next)
	}
	if snapshot := VectorClockSnapshot(); snapshot["america"] != 4 || snapshot["asia"] != 1 || snapshot["europa"] != 3 {
		t.Fatalf("unexpected vector %v", snapshot)
	}
	// Two writes that did not see each other are concurrent: neither dominates.
	if IsVectorDominant(`{"america":5,"europa":3}`, `{"asia":2,"europa":3}`) || IsVectorDominant(`{"asia":2,"europa":3}`, `{"america":5,"europa":3}`) {
		t.Fatal("concurrent writes must not dominate each other")
	}
}
