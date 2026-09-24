package services

import (
	"math"
	"strings"
)

// TSPRoute describes network edges, not a dated reservation itinerary.
type TSPRoute struct {
	Ruta   []string       `json:"ruta"`
	Costo  float64        `json:"costo"`
	Tiempo float64        `json:"tiempo"`
	Vuelos []RouteDetails `json:"vuelos"`
}

func CalculateTSP(ciudades []string, criterion, seatClass, region string) *TSPRoute {
	return CalculateTSPWithReturn(ciudades, criterion, seatClass, region, false)
}

func CalculateTSPWithReturn(ciudades []string, criterion, seatClass, region string, returnToOrigin bool) *TSPRoute {
	times, economy, first := getMatricesFromDB(region)
	return CalculateTSPFromMatrices(ciudades, criterion, seatClass, returnToOrigin, times, economy, first)
}

// CalculateTSPFromMatrices uses Held-Karp dynamic programming. Missing or zero
// fares are absent arcs even when optimizing travel time.
func CalculateTSPFromMatrices(ciudades []string, criterion, seatClass string, returnToOrigin bool,
	times, economy, first map[string]map[string]float64) *TSPRoute {
	n := len(ciudades)
	if n < 2 || n > 15 {
		return nil
	}
	seen := make(map[string]bool, n)
	for _, city := range ciudades {
		if city == "" || seen[city] {
			return nil
		}
		seen[city] = true
	}
	fares := economy
	if strings.EqualFold(seatClass, "VIP") || strings.EqualFold(seatClass, "FIRST") {
		fares = first
	}
	weight := func(i, j int) float64 {
		fare := fares[ciudades[i]][ciudades[j]]
		hours := times[ciudades[i]][ciudades[j]]
		if fare <= 0 || hours <= 0 {
			return math.Inf(1)
		}
		if strings.EqualFold(criterion, "TIME") {
			return hours
		}
		return fare
	}
	states := 1 << n
	dp := make([]float64, states*n)
	prev := make([]int, states*n)
	for i := range dp {
		dp[i] = math.Inf(1)
		prev[i] = -1
	}
	for i := 0; i < n; i++ {
		if !returnToOrigin || i == 0 {
			dp[(1<<i)*n+i] = 0
		}
	}
	for mask := 1; mask < states; mask++ {
		for last := 0; last < n; last++ {
			if mask&(1<<last) == 0 {
				continue
			}
			current := dp[mask*n+last]
			if math.IsInf(current, 1) {
				continue
			}
			for next := 0; next < n; next++ {
				if mask&(1<<next) != 0 {
					continue
				}
				arc := weight(last, next)
				if math.IsInf(arc, 1) {
					continue
				}
				nextMask := mask | (1 << next)
				index := nextMask*n + next
				if current+arc < dp[index] {
					dp[index] = current + arc
					prev[index] = last
				}
			}
		}
	}
	fullMask := states - 1
	best, end := math.Inf(1), -1
	for last := 0; last < n; last++ {
		cost := dp[fullMask*n+last]
		if returnToOrigin {
			cost += weight(last, 0)
		}
		if cost < best {
			best, end = cost, last
		}
	}
	if end < 0 || math.IsInf(best, 1) {
		return nil
	}
	indices := make([]int, n)
	mask, last := fullMask, end
	for position := n - 1; position >= 0; position-- {
		indices[position] = last
		nextLast := prev[mask*n+last]
		mask &^= 1 << last
		last = nextLast
	}
	if returnToOrigin {
		indices = append(indices, indices[0])
	}
	route := &TSPRoute{Ruta: make([]string, len(indices))}
	for i, index := range indices {
		route.Ruta[i] = ciudades[index]
		if i == 0 {
			continue
		}
		from, to := ciudades[indices[i-1]], ciudades[index]
		fare, hours := fares[from][to], times[from][to]
		route.Costo += fare
		route.Tiempo += hours
		route.Vuelos = append(route.Vuelos, RouteDetails{Salida: from, Llegada: to, Cost: fare, Time: hours})
	}
	return route
}
