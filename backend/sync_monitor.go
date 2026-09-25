package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"airres-api/db"
	"airres-api/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

// SyncSnapshot contains observations from actual database connections. Counts
// are unknown (nil), rather than zero, when a node cannot be queried.
type SyncNode struct {
	State      string     `json:"state"`
	Flights    *int64     `json:"flights"`
	Tickets    *int64     `json:"tickets"`
	Pending    *int64     `json:"pending,omitempty"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	LatencyMS  int64      `json:"latency_ms"`
}

type SyncEventRecord struct {
	ID      uint64    `json:"id"`
	At      time.Time `json:"at"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
}

type SyncSnapshot struct {
	ObservedAt time.Time           `json:"observed_at"`
	Nodes      map[string]SyncNode `json:"nodes"`
	Events     []SyncEventRecord   `json:"events"`
	ReadSource string              `json:"read_source"`
	Pending    int64               `json:"pending"`
	Drift      map[string]*int64   `json:"count_difference"`
}

var syncMonitor = struct {
	sync.RWMutex
	snapshot SyncSnapshot
	events   []SyncEventRecord
	nextID   uint64
}{events: make([]SyncEventRecord, 0, 100)}

func recordSyncEvent(kind, message string) {
	syncMonitor.Lock()
	defer syncMonitor.Unlock()
	syncMonitor.nextID++
	syncMonitor.events = append(syncMonitor.events, SyncEventRecord{
		ID: syncMonitor.nextID, At: time.Now().UTC(), Type: kind, Message: message,
	})
	if len(syncMonitor.events) > 100 {
		syncMonitor.events = syncMonitor.events[len(syncMonitor.events)-100:]
	}
}

func probePostgres(conn *gorm.DB, previous SyncNode) SyncNode {
	node := SyncNode{State: "DOWN", LastSeenAt: previous.LastSeenAt}
	started := time.Now()
	if !db.IsAvailable(conn) {
		return node
	}
	node.LatencyMS = time.Since(started).Milliseconds()
	now := time.Now().UTC()
	node.LastSeenAt = &now
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var flights, tickets, pending int64
	if conn.WithContext(ctx).Model(&models.Vuelo{}).Count(&flights).Error != nil ||
		conn.WithContext(ctx).Model(&models.Boleto{}).Count(&tickets).Error != nil ||
		conn.WithContext(ctx).Model(&models.SyncOutbox{}).Where("delivered_at = 0").Count(&pending).Error != nil {
		node.State = "DEGRADED"
		return node
	}
	node.State, node.Flights, node.Tickets, node.Pending = "UP", &flights, &tickets, &pending
	return node
}

func probeMongo(previous SyncNode) SyncNode {
	node := SyncNode{State: "DOWN", LastSeenAt: previous.LastSeenAt}
	started := time.Now()
	if !db.IsMongoAvailable() {
		return node
	}
	node.LatencyMS = time.Since(started).Milliseconds()
	now := time.Now().UTC()
	node.LastSeenAt = &now
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	flights, err := db.MongoDatabase.Collection("vuelos").CountDocuments(ctx, bson.M{})
	if err != nil {
		node.State = "DEGRADED"
		return node
	}
	tickets, err := db.MongoDatabase.Collection("boletos").CountDocuments(ctx, bson.M{})
	if err != nil {
		node.State = "DEGRADED"
		return node
	}
	node.State, node.Flights, node.Tickets = "UP", &flights, &tickets
	return node
}

func difference(a, b *int64) *int64 {
	if a == nil || b == nil {
		return nil
	}
	n := *a - *b
	if n < 0 {
		n = -n
	}
	return &n
}

func observeSync() {
	syncMonitor.RLock()
	previous := syncMonitor.snapshot
	syncMonitor.RUnlock()
	old := previous.Nodes
	if old == nil {
		old = map[string]SyncNode{}
	}
	nodes := map[string]SyncNode{
		"pg_am": probePostgres(db.PGAmerica, old["pg_am"]),
		"pg_eu": probePostgres(db.PGEuropaAsia, old["pg_eu"]),
		"mongo": probeMongo(old["mongo"]),
	}
	for key, node := range nodes {
		if was, exists := old[key]; exists && was.State != node.State {
			if node.State == "UP" {
				recordSyncEvent("recovered", key+" volvió a responder; se verifica la sincronización")
			} else {
				recordSyncEvent("failure", key+" cambió a "+node.State)
			}
		}
	}
	source := "none"
	if nodes["pg_am"].State == "UP" {
		source = "pg_am"
	} else if nodes["pg_eu"].State == "UP" {
		source = "pg_eu"
	} else if nodes["mongo"].State == "UP" {
		source = "mongo_snapshot"
	}
	var pending int64
	for _, key := range []string{"pg_am", "pg_eu"} {
		if value := nodes[key].Pending; value != nil {
			pending += *value
		}
	}
	drift := map[string]*int64{
		"am_eu_flights":    difference(nodes["pg_am"].Flights, nodes["pg_eu"].Flights),
		"am_eu_tickets":    difference(nodes["pg_am"].Tickets, nodes["pg_eu"].Tickets),
		"am_mongo_flights": difference(nodes["pg_am"].Flights, nodes["mongo"].Flights),
		"eu_mongo_flights": difference(nodes["pg_eu"].Flights, nodes["mongo"].Flights),
	}
	if previous.Nodes != nil && previous.Pending > 0 && pending == 0 && nodes["pg_am"].State == "UP" && nodes["pg_eu"].State == "UP" && nodes["mongo"].State == "UP" {
		recordSyncEvent("caught_up", "La cola de operaciones pendientes llegó a cero")
	}
	syncMonitor.Lock()
	syncMonitor.snapshot = SyncSnapshot{ObservedAt: time.Now().UTC(), Nodes: nodes, ReadSource: source, Pending: pending, Drift: drift}
	syncMonitor.Unlock()
}

func startSyncMonitor() {
	observeSync()
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		observeSync()
	}
}

func getSyncStatus(c *gin.Context) {
	syncMonitor.RLock()
	snapshot := syncMonitor.snapshot
	snapshot.Events = append([]SyncEventRecord{}, syncMonitor.events...)
	syncMonitor.RUnlock()
	if snapshot.Nodes == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Monitor iniciando"})
		return
	}
	requestedRegion := c.GetHeader("X-Region")
	if requestedRegion == "" {
		requestedRegion = "America"
	}
	conn, routedRegion := db.GetDBForCountry(requestedRegion)
	snapshot.ReadSource = "none"
	switch {
	case routedRegion == "Mongo" && db.IsMongoAvailable():
		snapshot.ReadSource = "mongo_snapshot"
	case conn == db.PGAmerica && conn != nil:
		snapshot.ReadSource = "pg_am"
	case conn == db.PGEuropaAsia && conn != nil:
		snapshot.ReadSource = "pg_eu"
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, snapshot)
}
