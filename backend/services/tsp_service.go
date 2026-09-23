package services

import (
	"math"
)

// TSPRoute represents a complete path and its cost/time
type TSPRoute struct {
	Ruta   []string       `json:"ruta"`
	Costo  float64        `json:"costo"`
	Tiempo float64        `json:"tiempo"`
	Vuelos []RouteDetails `json:"vuelos"`
}

// CalculateTSP calculates the shortest Hamiltonian path visiting all given cities exactly once.
// It is an open-ended path (does not return to origin).
func CalculateTSP(ciudades []string, criterion string, seatClass string, region string) *TSPRoute {
	if len(ciudades) < 2 {
		return nil
	}

	tiempos, preciosReg, preciosVip := getMatricesFromDB(region)
	
	var costMatrix map[string]map[string]float64
	if criterion == "TIME" {
		costMatrix = tiempos
	} else {
		if seatClass == "VIP" {
			costMatrix = preciosVip
		} else {
			costMatrix = preciosReg
		}
	}

	var bestRoute []string
	bestVal := math.MaxFloat64

	var permute func(arr []string, l, r int)
	permute = func(arr []string, l, r int) {
		if l == r {
			// Evaluate current permutation
			valid := true
			currentVal := 0.0
			for i := 0; i < len(arr)-1; i++ {
				from := arr[i]
				to := arr[i+1]
				
				if costMatrix[from] == nil {
					valid = false
					break
				}
				val := costMatrix[from][to]
				// 0 means no route, unless from == to (which shouldn't happen here)
				if val == 0 && from != to {
					valid = false
					break
				}
				currentVal += val
			}
			
			if valid && currentVal < bestVal {
				bestVal = currentVal
				bestRoute = make([]string, len(arr))
				copy(bestRoute, arr)
			}
		} else {
			for i := l; i <= r; i++ {
				arr[l], arr[i] = arr[i], arr[l]
				permute(arr, l+1, r)
				arr[l], arr[i] = arr[i], arr[l] // backtrack
			}
		}
	}

	permute(ciudades, 0, len(ciudades)-1)

	if bestVal == math.MaxFloat64 || len(bestRoute) == 0 {
		return nil
	}

	resp := &TSPRoute{
		Ruta: bestRoute,
	}
	
	for i := 0; i < len(bestRoute)-1; i++ {
		from := bestRoute[i]
		to := bestRoute[i+1]
		
		t := tiempos[from][to]
		c := 0.0
		if seatClass == "VIP" {
			c = preciosVip[from][to]
		} else {
			c = preciosReg[from][to]
		}
		
		resp.Tiempo += t
		resp.Costo += c
		resp.Vuelos = append(resp.Vuelos, RouteDetails{
			Salida:  from,
			Llegada: to,
			Cost:    c,
			Time:    t,
		})
	}
	
	return resp
}
