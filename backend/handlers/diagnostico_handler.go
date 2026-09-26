package handlers

import (
	"net/http"

	"airres-api/data"
	"airres-api/db"
	"airres-api/models"
	"github.com/gin-gonic/gin"
)

func GetContinuityReport(c *gin.Context) {
	if db.PGAmerica == nil || db.PGEuropaAsia == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Bases regionales no disponibles"})
		return
	}
	byID := make(map[uint]models.Vuelo)
	var american, other []models.Vuelo
	if err := db.PGAmerica.Find(&american).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if err := db.PGEuropaAsia.Find(&other).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	for _, flight := range append(american, other...) {
		if old, ok := byID[flight.ID]; !ok || flight.LamportClock > old.LamportClock {
			byID[flight.ID] = flight
		}
	}
	flights := make([]models.Vuelo, 0, len(byID))
	for _, flight := range byID {
		flights = append(flights, flight)
	}
	// Both regions store the same positioning schedule; merge in case one missed it.
	var americanFerries, otherFerries []models.Reposicionamiento
	if err := db.PGAmerica.Find(&americanFerries).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if err := db.PGEuropaAsia.Find(&otherFerries).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	ferryByFlight := make(map[uint]models.Reposicionamiento)
	for _, ferry := range append(americanFerries, otherFerries...) {
		ferryByFlight[ferry.IDVueloSiguiente] = ferry
	}
	ferries := make([]models.Reposicionamiento, 0, len(ferryByFlight))
	for _, ferry := range ferryByFlight {
		ferries = append(ferries, ferry)
	}
	c.JSON(http.StatusOK, data.AnalyzeContinuityWithRepositioning(flights, ferries))
}
