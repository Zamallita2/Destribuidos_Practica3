package services

import (
	"sync"
)

// lamportClock struct to hold the clock safely across concurrent goroutines
type lamportClock struct {
	mu    sync.Mutex
	clock int64
}

// GlobalLamportClock is the single instance of the clock for the node
var GlobalLamportClock = &lamportClock{
	clock: 0,
}

// Tick increments the Lamport clock by 1 and returns the new value.
// Call this BEFORE performing any local event (like creating or updating a record).
func (lc *lamportClock) Tick() int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.clock++
	return lc.clock
}

// UpdateClock updates the internal clock when receiving a message from another node.
// Rule: L = max(L, incoming_L) + 1
func (lc *lamportClock) UpdateClock(incomingClock int64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if incomingClock > lc.clock {
		lc.clock = incomingClock
	}
	lc.clock++
}

// GetCurrent returns the current clock value without incrementing it
func (lc *lamportClock) GetCurrent() int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.clock
}
