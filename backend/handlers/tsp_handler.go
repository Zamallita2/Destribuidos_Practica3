package handlers

import (
	"net/http"

	"airres-api/services"

	"github.com/gin-gonic/gin"
)

type TSPRequest struct {
	Ciudades       []string `json:"ciudades"`
	Criterio       string   `json:"criterio"`   // "TIME" or "COST"
	SeatClass      string   `json:"seat_class"` // "REGULAR" or "VIP"
	ReturnToOrigin bool     `json:"return_to_origin"`
}

// GetTSPRoute returns the shortest Hamiltonian path visiting all given cities.
func GetTSPRoute(c *gin.Context) {
	var payload TSPRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	if len(payload.Ciudades) < 2 || len(payload.Ciudades) > 15 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe proporcionar entre 2 y 15 ciudades distintas"})
		return
	}
	seen := make(map[string]bool, len(payload.Ciudades))
	for _, city := range payload.Ciudades {
		if seen[city] || city == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Las ciudades deben ser distintas y no vacías"})
			return
		}
		seen[city] = true
	}
	if payload.Criterio != "TIME" && payload.Criterio != "COST" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Criterio debe ser TIME o COST"})
		return
	}
	if payload.SeatClass != "REGULAR" && payload.SeatClass != "VIP" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seat_class debe ser REGULAR o VIP"})
		return
	}

	region := c.GetHeader("X-Region")
	if region == "" {
		region = "America"
	}

	route := services.CalculateTSPWithReturn(payload.Ciudades, payload.Criterio, payload.SeatClass, region, payload.ReturnToOrigin)

	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se pudo encontrar una ruta válida que conecte todas las ciudades especificadas"})
		return
	}

	c.JSON(http.StatusOK, route)
}
