package services

import (
	"encoding/json"
	"os"
	"sync"
)

type VectorClock map[string]int64

var (
	GlobalVectorClock = make(VectorClock)
	vcMutex           sync.RWMutex
	NodeID            string
)

func init() {
	NodeID = os.Getenv("NODE_ID")
	if NodeID == "" {
		NodeID = "backend"
	}
	GlobalVectorClock[NodeID] = 0
}

// TickVectorClock increments the vector clock for this node and returns the serialized JSON string
func TickVectorClock() string {
	vcMutex.Lock()
	defer vcMutex.Unlock()
	GlobalVectorClock[NodeID]++
	b, _ := json.Marshal(GlobalVectorClock)
	return string(b)
}

// UpdateVectorClock merges an incoming vector clock with the global one
func UpdateVectorClock(incoming string) {
	if incoming == "" {
		return
	}
	var inClock VectorClock
	if err := json.Unmarshal([]byte(incoming), &inClock); err != nil {
		return
	}

	vcMutex.Lock()
	defer vcMutex.Unlock()
	for k, v := range inClock {
		if current, exists := GlobalVectorClock[k]; !exists || v > current {
			GlobalVectorClock[k] = v
		}
	}
}

// IsVectorDominant returns true if clockA is strictly newer than clockB
// Returns false if they are concurrent or if clockB is newer
func IsVectorDominant(clockA, clockB string) bool {
	if clockA == "" {
		return false
	}
	if clockB == "" {
		return true
	}

	var a, b VectorClock
	if err := json.Unmarshal([]byte(clockA), &a); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(clockB), &b); err != nil {
		return true
	}

	isStrictlyGreater := false
	for k, valA := range a {
		valB := b[k]
		if valA < valB {
			return false
		}
		if valA > valB {
			isStrictlyGreater = true
		}
	}

	for k, valB := range b {
		valA := a[k]
		if valA < valB {
			return false
		}
	}

	return isStrictlyGreater
}

// ShouldApplyVersion preserves causal order. Concurrent versions use a
// deterministic Lamport/node ordering; booking conflicts are handled by
// the single writer and unique seat constraint instead.
func ShouldApplyVersion(incomingVector, existingVector string, incomingLamport, existingLamport int64,
	incomingNode, existingNode string) bool {
	if incomingVector != "" && existingVector != "" {
		if IsVectorDominant(incomingVector, existingVector) {
			return true
		}
		if IsVectorDominant(existingVector, incomingVector) {
			return false
		}
	}
	if incomingLamport != existingLamport {
		return incomingLamport > existingLamport
	}
	return incomingNode > existingNode
}
