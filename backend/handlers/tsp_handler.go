package handlers

import (
	"net/http"

	"airres-api/services"

	"github.com/gin-gonic/gin"
)

type TSPRequest struct {
	Ciudades  []string `json:"ciudades"`
	Criterio  string   `json:"criterio"` // "TIME" or "COST"
	SeatClass string   `json:"seat_class"` // "REGULAR" or "VIP"
}

// GetTSPRoute returns the shortest Hamiltonian path visiting all given cities.
func GetTSPRoute(c *gin.Context) {
	var payload TSPRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	if len(payload.Ciudades) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar al menos 2 ciudades"})
		return
	}

	region := c.GetHeader("X-Region")
	if region == "" {
		region = "America"
	}

	route := services.CalculateTSP(payload.Ciudades, payload.Criterio, payload.SeatClass, region)
	
	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se pudo encontrar una ruta válida que conecte todas las ciudades especificadas"})
		return
	}

	c.JSON(http.StatusOK, route)
}
