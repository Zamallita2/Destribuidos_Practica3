package handlers

import (
	"net/http"

	"airres-api/services"

	"github.com/gin-gonic/gin"
)

// GetSugerencias returns top 3 shortest paths using Dijkstra (Yen's K-Shortest Path)
func GetSugerencias(c *gin.Context) {
	criterio := c.Param("criterio")
	origen := c.Query("origen")
	destino := c.Query("destino")
	clase := c.Query("clase") // "regular" or "vip"

	if origen == "" || destino == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Origen y Destino son requeridos"})
		return
	}

	if clase == "" {
		clase = "regular"
	}

	if criterio != "costo" && criterio != "tiempo" {
		criterio = "tiempo"
	}

	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}

	rutas := services.FindTop3Paths(origen, destino, criterio, clase, tag)

	// If it doesn't find any, return empty list
	if len(rutas) == 0 {
		c.JSON(http.StatusOK, []services.SuggestedRoute{})
		return
	}

	c.JSON(http.StatusOK, rutas)
}
