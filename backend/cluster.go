package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"airres-api/services"

	"github.com/gin-gonic/gin"
)

var nodeStartedAt = time.Now().UTC()

type nodeInfo struct {
	NodeID      string               `json:"node_id"`
	Primary     bool                 `json:"primary"`
	TimeZone    string               `json:"time_zone"`
	LocalTime   string               `json:"local_time"`
	StartedAt   time.Time            `json:"started_at"`
	Lamport     int64                `json:"lamport_clock"`
	VectorClock services.VectorClock `json:"vector_clock"`
}

type peerStatus struct {
	nodeInfo
	Up        bool   `json:"up"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

func currentNodeInfo() nodeInfo {
	now := time.Now()
	return nodeInfo{
		NodeID: services.NodeID, Primary: os.Getenv("NODE_PRIMARY") != "false",
		TimeZone: time.Local.String(), LocalTime: now.Format("2006-01-02 15:04:05 MST"),
		StartedAt: nodeStartedAt, Lamport: services.GlobalLamportClock.GetCurrent(),
		VectorClock: services.VectorClockSnapshot(),
	}
}

// getNodeInfo reports this API server's identity and logical clocks.
func getNodeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, currentNodeInfo())
}

// clusterPeers parses CLUSTER_PEERS="america=http://backend_am:8080,europa=...".
func clusterPeers() map[string]string {
	peers := map[string]string{}
	for _, entry := range strings.Split(os.Getenv("CLUSTER_PEERS"), ",") {
		name, url, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if ok && name != "" && url != "" {
			peers[name] = strings.TrimRight(url, "/")
		}
	}
	return peers
}

// getClusterInfo asks every API server for its clocks so the synchronization
// panel can show the three vector clocks side by side.
func getClusterInfo(c *gin.Context) {
	peers := clusterPeers()
	results := make([]peerStatus, 0, len(peers)+1)
	var mu sync.Mutex
	var wg sync.WaitGroup
	client := http.Client{Timeout: 1500 * time.Millisecond}
	self := false
	for name, url := range peers {
		if name == services.NodeID {
			self = true
			results = append(results, peerStatus{nodeInfo: currentNodeInfo(), Up: true})
			continue
		}
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			status := peerStatus{nodeInfo: nodeInfo{NodeID: name}}
			started := time.Now()
			ctx, cancel := context.WithTimeout(c.Request.Context(), 1500*time.Millisecond)
			defer cancel()
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, url+"/api/node", nil)
			response, err := client.Do(request)
			if err == nil {
				defer response.Body.Close()
				if response.StatusCode == http.StatusOK && json.NewDecoder(response.Body).Decode(&status.nodeInfo) == nil {
					status.Up = true
				} else {
					err = context.DeadlineExceeded
				}
			}
			status.LatencyMS = time.Since(started).Milliseconds()
			if err != nil {
				status.Error = "sin_respuesta"
			}
			mu.Lock()
			results = append(results, status)
			mu.Unlock()
		}(name, url)
	}
	wg.Wait()
	if !self {
		results = append(results, peerStatus{nodeInfo: currentNodeInfo(), Up: true})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].NodeID < results[j].NodeID })
	c.JSON(http.StatusOK, gin.H{"served_by": services.NodeID, "nodes": results})
}
