package db

import (
	"errors"

	"airres-api/models"
	"gorm.io/gorm"
)

// GetDBForFlightWrite makes the origin region the sole writer for one flight.
// Replicas may serve reads, but a client-supplied region cannot choose a
// different writer for the same seat.
func GetDBForFlightWrite(flightID uint) (*gorm.DB, models.Vuelo, error) {
	var flight models.Vuelo
	preferred, backup := PGAmerica, PGEuropaAsia
	if flightID >= 1000000000 {
		preferred, backup = PGEuropaAsia, PGAmerica
	}
	for _, conn := range []*gorm.DB{preferred, backup} {
		if !IsAvailable(conn) {
			continue
		}
		if err := conn.First(&flight, flightID).Error; err == nil {
			return conn, flight, nil
		}
	}
	return nil, flight, errors.New("vuelo_no_encontrado_o_nodos_no_disponibles")
}

func GetDBForTicketWrite(ticketID uint) (*gorm.DB, models.Boleto, error) {
	owner, backup := PGAmerica, PGEuropaAsia
	if ticketID >= 1000000000 {
		owner, backup = PGEuropaAsia, PGAmerica
	}
	var ticket models.Boleto
	for _, conn := range []*gorm.DB{owner, backup} {
		if !IsAvailable(conn) {
			continue
		}
		if err := conn.First(&ticket, ticketID).Error; err == nil {
			return conn, ticket, nil
		}
	}
	return nil, ticket, errors.New("boleto_no_encontrado_o_nodos_no_disponibles")
}

func GetDBForOriginWrite(originID uint) (*gorm.DB, error) {
	var city models.Ciudad
	lookup := PGAmerica
	if !IsAvailable(lookup) {
		lookup = PGEuropaAsia
	}
	if !IsAvailable(lookup) {
		return nil, errors.New("no_database_available")
	}
	if err := lookup.First(&city, originID).Error; err != nil {
		return nil, err
	}
	if city.Region == "America" {
		if IsAvailable(PGAmerica) {
			return PGAmerica, nil
		}
		if IsAvailable(PGEuropaAsia) {
			return PGEuropaAsia, nil
		}
		return nil, errors.New("nodo_propietario_no_disponible")
	}
	if IsAvailable(PGEuropaAsia) {
		return PGEuropaAsia, nil
	}
	if IsAvailable(PGAmerica) {
		return PGAmerica, nil
	}
	return nil, errors.New("nodo_propietario_no_disponible")
}
