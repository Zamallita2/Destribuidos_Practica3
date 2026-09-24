package services

import (
	"encoding/json"
	"errors"
	"strings"

	"airres-api/models"
	"gorm.io/gorm"
)

// RouteFare returns the fare for one class and one direction. JSON nulls
// decode as zero, which is not a sellable fare.
func RouteFare(origin, destination, class string, economy, first map[string]map[string]float64) (float64, bool) {
	var fare float64
	switch strings.ToUpper(class) {
	case "REGULAR", "ECONOMY":
		fare = economy[origin][destination]
	case "VIP", "FIRST":
		fare = first[origin][destination]
	default:
		return 0, false
	}
	return fare, fare > 0 && origin != destination
}

func FareForFlight(tx *gorm.DB, flight *models.Vuelo, class string) (float64, error) {
	var origin, destination models.Ciudad
	if err := tx.First(&origin, flight.IDOrigen).Error; err != nil {
		return 0, err
	}
	if err := tx.First(&destination, flight.IDDestino).Error; err != nil {
		return 0, err
	}
	var pricing models.Precios
	if err := tx.First(&pricing).Error; err != nil {
		return 0, err
	}
	var economy, first map[string]map[string]float64
	if err := json.Unmarshal(pricing.MatrizPreciosRegular, &economy); err != nil {
		return 0, err
	}
	if err := json.Unmarshal(pricing.MatrizPreciosVip, &first); err != nil {
		return 0, err
	}
	fare, ok := RouteFare(origin.Codigo, destination.Codigo, class, economy, first)
	if !ok {
		return 0, errors.New("clase_no_disponible")
	}
	return fare, nil
}

func DurationForFlight(tx *gorm.DB, flight *models.Vuelo) (int64, error) {
	var origin, destination models.Ciudad
	if err := tx.First(&origin, flight.IDOrigen).Error; err != nil {
		return 0, err
	}
	if err := tx.First(&destination, flight.IDDestino).Error; err != nil {
		return 0, err
	}
	var details models.DetallesVuelos
	if err := tx.First(&details).Error; err != nil {
		return 0, err
	}
	var times map[string]map[string]float64
	if err := json.Unmarshal(details.MatrizTiempos, &times); err != nil {
		return 0, err
	}
	hours := times[origin.Codigo][destination.Codigo]
	if hours <= 0 || origin.Codigo == destination.Codigo {
		return 0, errors.New("ruta_sin_duracion")
	}
	return int64(hours * 3600), nil
}
