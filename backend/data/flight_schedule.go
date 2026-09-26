package data

import (
	"sort"

	"airres-api/models"
)

// FlightStatusAt assigns imported flights from their scheduled interval.
func FlightStatusAt(departure, arrival, now int64) uint {
	if arrival <= now {
		return 5
	}
	if departure <= now {
		return 4
	}
	return 1
}

type aircraftState struct {
	location uint  // 0 until the aircraft is first positioned
	freeAt   int64 // arrival time of its last movement
}

// AssignAircraft schedules flights in chronological order so that no aircraft
// is in two places at once or appears at an airport it never flew to. Each
// flight keeps its CSV aircraft when possible; otherwise it takes an aircraft
// already waiting at the origin, then one that has not flown yet, and finally
// one that can reach the origin in time through an empty positioning flight.
// hours returns the travel time between two cities, or 0 when there is no route.
func AssignAircraft(flights []models.Vuelo, planeIDs []uint, hours func(from, to uint) float64) (
	accepted []models.Vuelo, ferries []models.Reposicionamiento, rejected []models.Vuelo) {
	ordered := append([]models.Vuelo(nil), flights...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SalidaProgramada != ordered[j].SalidaProgramada {
			return ordered[i].SalidaProgramada < ordered[j].SalidaProgramada
		}
		return ordered[i].ID < ordered[j].ID
	})
	planes := append([]uint(nil), planeIDs...)
	sort.Slice(planes, func(i, j int) bool { return planes[i] < planes[j] })
	state := make(map[uint]*aircraftState, len(planes))
	for _, id := range planes {
		state[id] = &aircraftState{}
	}
	ferrySeconds := func(plane uint, to uint) int64 {
		s := state[plane]
		if s.location == 0 || s.location == to {
			return 0
		}
		if h := hours(s.location, to); h > 0 {
			return int64(h * 3600)
		}
		return -1
	}

	for _, flight := range ordered {
		departure := flight.SalidaProgramada
		waiting := func(plane uint) bool {
			s := state[plane]
			return s.location == flight.IDOrigen && s.freeAt <= departure
		}
		unused := func(plane uint) bool { return state[plane].location == 0 }
		reachable := func(plane uint) bool {
			seconds := ferrySeconds(plane, flight.IDOrigen)
			return seconds > 0 && state[plane].freeAt+seconds <= departure
		}

		chosen := uint(0)
		for _, fits := range []func(uint) bool{waiting, unused, reachable} {
			if _, known := state[flight.IDAvion]; known && fits(flight.IDAvion) {
				chosen = flight.IDAvion
				break
			}
			for _, plane := range planes {
				if fits(plane) {
					chosen = plane
					break
				}
			}
			if chosen != 0 {
				break
			}
		}
		if chosen == 0 {
			rejected = append(rejected, flight)
			continue
		}

		if seconds := ferrySeconds(chosen, flight.IDOrigen); seconds > 0 {
			ferries = append(ferries, models.Reposicionamiento{
				IDVueloSiguiente: flight.ID, IDAvion: chosen,
				IDOrigen: state[chosen].location, IDDestino: flight.IDOrigen,
				SalidaProgramada: departure - seconds, LlegadaProgramada: departure,
			})
		}
		flight.IDAvion = chosen
		state[chosen].location = flight.IDDestino
		state[chosen].freeAt = flight.LlegadaProgramada
		accepted = append(accepted, flight)
	}
	return accepted, ferries, rejected
}
