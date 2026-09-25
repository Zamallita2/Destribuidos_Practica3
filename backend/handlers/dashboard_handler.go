package handlers

import (
	"net/http"
	"strconv"

	"airres-api/db"
	"airres-api/models"
	"airres-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type dashboardTotals struct {
	Vuelos              int          `json:"vuelos"`
	Vendidos            int          `json:"vendidos"`
	Reservados          int          `json:"reservados"`
	Disponibles         int          `json:"disponibles"`
	IngresosPrimera     float64      `json:"ingresos_primera"`
	IngresosTuristica   float64      `json:"ingresos_turistica"`
	EstadosVuelo        map[uint]int `json:"estados_vuelo"`
	ManifestosGenerados int          `json:"manifiestos_generados"`
}

func GetDashboardSummary(c *gin.Context) {
	am, eu := db.PGAmerica, db.PGEuropaAsia
	if !db.IsAvailable(am) {
		am = eu
	}
	if !db.IsAvailable(eu) {
		eu = am
	}
	if !db.IsAvailable(am) || !db.IsAvailable(eu) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Bases regionales no disponibles"})
		return
	}
	var americanFlights, otherFlights []models.Vuelo
	var americanTickets, otherTickets []models.Boleto
	var manifestsAM, manifestsEU []models.OcupacionVuelo
	for _, err := range []error{
		am.Where("id < ?", 1000000000).Find(&americanFlights).Error,
		eu.Where("id >= ?", 1000000000).Find(&otherFlights).Error,
		am.Where("id_boleto < ?", 1000000000).Find(&americanTickets).Error,
		eu.Where("id_boleto >= ?", 1000000000).Find(&otherTickets).Error,
		am.Where("id_vuelo < ?", 1000000000).Select("id_vuelo, eligible_seats, sold_count, reserved_count, first_income, economy_income").Find(&manifestsAM).Error,
		eu.Where("id_vuelo >= ?", 1000000000).Select("id_vuelo, eligible_seats, sold_count, reserved_count, first_income, economy_income").Find(&manifestsEU).Error,
	} {
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
	}
	summary := dashboardTotals{EstadosVuelo: make(map[uint]int)}
	byID := make(map[uint]models.Vuelo)
	for _, flight := range append(americanFlights, otherFlights...) {
		if old, found := byID[flight.ID]; !found || flight.LamportClock > old.LamportClock {
			byID[flight.ID] = flight
		}
	}
	summary.Vuelos = len(byID)
	for _, flight := range byID {
		summary.EstadosVuelo[flight.IDEstadoVuelo]++
	}
	for _, manifest := range append(manifestsAM, manifestsEU...) {
		summary.ManifestosGenerados++
		summary.Vendidos += manifest.SoldCount
		summary.Reservados += manifest.ReservedCount
		summary.Disponibles += manifest.EligibleSeats - manifest.SoldCount - manifest.ReservedCount
		summary.IngresosPrimera += manifest.FirstIncome
		summary.IngresosTuristica += manifest.EconomyIncome
	}
	for _, ticket := range append(americanTickets, otherTickets...) {
		switch ticket.Estado {
		case "SALED":
			summary.Vendidos++
			if ticket.Clase == "VIP" {
				summary.IngresosPrimera += ticket.Costo
			} else {
				summary.IngresosTuristica += ticket.Costo
			}
			summary.Disponibles--
		case "RESERVED", "REFUNDED":
			summary.Reservados++
			summary.Disponibles--
		}
	}
	if summary.Disponibles < 0 {
		summary.Disponibles = 0
	}
	c.JSON(http.StatusOK, summary)
}

func GetFlightDashboard(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	owner, flight, err := db.GetDBForFlightWrite(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
		return
	}
	var manifest models.OcupacionVuelo
	if err := owner.Transaction(func(tx *gorm.DB) error {
		var ensureErr error
		manifest, ensureErr = services.EnsureFlightOccupancy(tx, flight)
		return ensureErr
	}); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	var tickets []models.Boleto
	if err := owner.Where("id_vuelo = ? AND estado IN ?", id,
		[]string{"RESERVED", "SALED", "REFUNDED"}).Find(&tickets).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	sold, reserved := manifest.SoldCount, manifest.ReservedCount
	available := manifest.EligibleSeats - sold - reserved
	firstIncome, economyIncome := manifest.FirstIncome, manifest.EconomyIncome
	for _, ticket := range tickets {
		if ticket.Estado == "SALED" {
			sold++
			if ticket.Clase == "VIP" {
				firstIncome += ticket.Costo
			} else {
				economyIncome += ticket.Costo
			}
		} else {
			reserved++
		}
		available--
	}
	if available < 0 {
		available = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"id_vuelo": flight.ID, "estado_vuelo": flight.IDEstadoVuelo,
		"capacidad": manifest.EligibleSeats, "vendidos": sold, "reservados": reserved,
		"disponibles": available, "ingresos_primera": firstIncome,
		"ingresos_turistica": economyIncome,
	})
}
