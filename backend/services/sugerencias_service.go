package services

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"airres-api/db"
	"airres-api/models"

	"go.mongodb.org/mongo-driver/bson"
)

type RouteDetails struct {
	Salida  string  `json:"salida"`
	Llegada string  `json:"llegada"`
	Cost    float64 `json:"cost"`
	Time    float64 `json:"time"`
}

type SuggestedRoute struct {
	Ruta   []string       `json:"ruta"`
	Costo  float64        `json:"costo"`
	Tiempo float64        `json:"tiempo"`
	Vuelos []RouteDetails `json:"vuelos"`
}

func getMatricesFromDB(region string) (map[string]map[string]float64, map[string]map[string]float64, map[string]map[string]float64) {
	dbConn, reg := db.GetDBForCountry(region)

	var detalles models.DetallesVuelos
	var precios models.Precios

	if reg == "Mongo" && db.MongoDatabase != nil {
		collDetalles := db.MongoDatabase.Collection("detalles_vuelos")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var resDetalles map[string]interface{}
		collDetalles.FindOne(ctx, bson.M{}).Decode(&resDetalles)
		if val, ok := resDetalles["matriz_tiempos"]; ok {
			jb, _ := json.Marshal(val)
			json.Unmarshal(jb, &detalles.MatrizTiempos)
		}

		collPrecios := db.MongoDatabase.Collection("precios")
		var resPrecios map[string]interface{}
		collPrecios.FindOne(ctx, bson.M{}).Decode(&resPrecios)
		if val, ok := resPrecios["matriz_precios_regular"]; ok {
			jb, _ := json.Marshal(val)
			json.Unmarshal(jb, &precios.MatrizPreciosRegular)
		}
		if val, ok := resPrecios["matriz_precios_vip"]; ok {
			jb, _ := json.Marshal(val)
			json.Unmarshal(jb, &precios.MatrizPreciosVip)
		}
	} else if dbConn != nil {
		dbConn.First(&detalles)
		dbConn.First(&precios)
	}

	var tMatrix, pMatrixReg, pMatrixVip map[string]map[string]float64
	json.Unmarshal(detalles.MatrizTiempos, &tMatrix)
	json.Unmarshal(precios.MatrizPreciosRegular, &pMatrixReg)
	json.Unmarshal(precios.MatrizPreciosVip, &pMatrixVip)

	return tMatrix, pMatrixReg, pMatrixVip
}

// FindTop3Paths finds the top 3 routes using Yen's K-Shortest Paths algorithm
func FindTop3Paths(origen, destino, criterio, clase, tag string) []SuggestedRoute {
	tMatrix, pMatrixReg, pMatrixVip := getMatricesFromDB(tag)
	if origen == destino || tMatrix[origen] == nil || tMatrix[destino] == nil {
		return []SuggestedRoute{}
	}

	// Determine the primary weights (cost) and secondary weights
	var primaryMatrix map[string]map[string]float64
	if clase == "vip" {
		primaryMatrix = pMatrixVip
	} else {
		primaryMatrix = pMatrixReg
	}

	// We define 'weight' function
	getWeight := func(u, v string) float64 {
		if u == v {
			return 0
		}

		validCost := primaryMatrix[u][v] > 0 && tMatrix[u][v] > 0

		// Wait, sometimes matrix might have null -> which unmarshals to 0.
		// If cost is 0 in a JSON map but nodes are different, it means null (no direct flight).
		if !validCost {
			return math.Inf(1)
		}

		if criterio == "costo" {
			return primaryMatrix[u][v]
		}
		// If criterio == tiempo, we use time matrix but only if cost is valid
		if !validCost {
			return math.Inf(1) // Can't go if we can't afford / no direct flight in that class
		}
		return tMatrix[u][v]
	}

	paths := yenKSP(getWeight, tMatrix, primaryMatrix, origen, destino, 3)
	return paths
}

type dijkstraState struct {
	node string
	dist float64
	prev string
}

func yenKSP(
	weightFunc func(u, v string) float64,
	tMatrix map[string]map[string]float64,
	pMatrix map[string]map[string]float64,
	source, sink string,
	K int,
) []SuggestedRoute {
	// A is the list of shortest paths
	var A [][]string
	var ACosts []float64

	// Function to calculate pure Dijkstra given removed nodes and edges
	dijkstra := func(src, dst string, removedNodes map[string]bool, removedEdges map[string]map[string]bool) ([]string, float64) {
		dist := make(map[string]float64)
		prev := make(map[string]string)
		unvisited := make(map[string]bool)

		if tMatrix == nil {
			return nil, math.Inf(1)
		}

		for u := range tMatrix {
			if !removedNodes[u] {
				dist[u] = math.Inf(1)
				unvisited[u] = true
			}
		}

		dist[src] = 0

		for len(unvisited) > 0 {
			var u string
			minDist := math.Inf(1)
			for n := range unvisited {
				if dist[n] < minDist {
					minDist = dist[n]
					u = n
				}
			}

			if u == "" || u == dst {
				break
			}
			delete(unvisited, u)

			for v := range tMatrix[u] {
				if removedNodes[v] {
					continue
				}
				if removedEdges[u] != nil && removedEdges[u][v] {
					continue
				}

				w := weightFunc(u, v)
				if math.IsInf(w, 1) {
					continue
				}

				alt := dist[u] + w
				if alt < dist[v] {
					dist[v] = alt
					prev[v] = u
				}
			}
		}

		if math.IsInf(dist[dst], 1) {
			return nil, math.Inf(1)
		}

		path := []string{}
		curr := dst
		for curr != "" {
			path = append([]string{curr}, path...)
			if curr == src {
				break
			}
			curr = prev[curr]
		}
		return path, dist[dst]
	}

	shortestPath, shortestCost := dijkstra(source, sink, make(map[string]bool), make(map[string]map[string]bool))
	if shortestPath == nil {
		return []SuggestedRoute{}
	}

	A = append(A, shortestPath)
	ACosts = append(ACosts, shortestCost)

	// B represents potential shortest paths
	type bPath struct {
		path []string
		cost float64
	}
	var B []bPath

	for k := 1; k < K; k++ {
		prevPath := A[k-1]
		for i := 0; i < len(prevPath)-1; i++ {
			spurNode := prevPath[i]
			rootPath := prevPath[:i+1]

			removedEdges := make(map[string]map[string]bool)
			for _, p := range A {
				// if rootPath matches
				match := true
				if len(p) > i {
					for x := 0; x <= i; x++ {
						if p[x] != rootPath[x] {
							match = false
							break
						}
					}
				} else {
					match = false
				}

				if match && i+1 < len(p) {
					u, v := p[i], p[i+1]
					if removedEdges[u] == nil {
						removedEdges[u] = make(map[string]bool)
					}
					removedEdges[u][v] = true
				}
			}

			removedNodes := make(map[string]bool)
			for _, rootNode := range rootPath {
				if rootNode != spurNode {
					removedNodes[rootNode] = true
				}
			}

			spurPath, spurCost := dijkstra(spurNode, sink, removedNodes, removedEdges)
			if spurPath != nil {
				// totalPath
				totalCost := 0.0
				totalPath := make([]string, 0)
				for j := 0; j < len(rootPath)-1; j++ {
					totalCost += weightFunc(rootPath[j], rootPath[j+1])
					totalPath = append(totalPath, rootPath[j])
				}
				totalCost += spurCost
				totalPath = append(totalPath, spurPath...)

				// add to B if not in B and not in A
				inA := false
				for _, aP := range A {
					if pathsEqual(aP, totalPath) {
						inA = true
						break
					}
				}
				if !inA {
					inB := false
					for _, bP := range B {
						if pathsEqual(bP.path, totalPath) {
							inB = true
							break
						}
					}
					if !inB {
						B = append(B, bPath{path: totalPath, cost: totalCost})
					}
				}
			}
		}

		if len(B) == 0 {
			break
		}

		// Sort B by cost
		sort.Slice(B, func(i, j int) bool {
			return B[i].cost < B[j].cost
		})

		A = append(A, B[0].path)
		ACosts = append(ACosts, B[0].cost)
		B = B[1:]
	}

	// Format results
	var results []SuggestedRoute
	for _, p := range A {
		tCost := 0.0
		tTime := 0.0
		var vuelos []RouteDetails

		for x := 0; x < len(p)-1; x++ {
			u, v := p[x], p[x+1]
			c := pMatrix[u][v]
			t := tMatrix[u][v]

			tCost += c
			tTime += t

			vuelos = append(vuelos, RouteDetails{
				Salida:  u,
				Llegada: v,
				Cost:    c,
				Time:    t,
			})
		}
		results = append(results, SuggestedRoute{
			Ruta:   p,
			Costo:  tCost,
			Tiempo: tTime,
			Vuelos: vuelos,
		})
	}

	return results
}

func pathsEqual(p1, p2 []string) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			return false
		}
	}
	return true
}
