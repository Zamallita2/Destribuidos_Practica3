package services

import (
	"errors"
	"time"

	"airres-api/models"
)

func ValidateBooking(flight models.Vuelo, seat models.Asiento, desiredState string, now time.Time) error {
	if desiredState != "RESERVED" && desiredState != "SALED" {
		return errors.New("estado_invalido")
	}
	if flight.IDAvion != seat.IDAvion {
		return errors.New("asiento_de_otro_avion")
	}
	if seat.Estado != "AVAILABLE" {
		return errors.New("asiento_no_disponible")
	}
	if flight.SalidaProgramada <= now.Unix() {
		return errors.New("vuelo_pasado")
	}
	if flight.IDEstadoVuelo != 1 && flight.IDEstadoVuelo != 8 {
		return errors.New("vuelo_cerrado")
	}
	return nil
}
