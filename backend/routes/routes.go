package routes

import (
	"github.com/gin-gonic/gin"
	"airres-api/handlers"
)

func SetupRoutes(r *gin.Engine) {
	// API Group
	api := r.Group("/api")

	// Verify DB connectivity endpoint
	api.GET("/verify", func(c *gin.Context) {
		country := c.GetHeader("X-User-Country")
		if country == "" {
			country = "Unknown"
		}
		
		c.JSON(200, gin.H{
			"status": "success",
			"country": country,
			"message": "Connected to cluster",
		})
	})

	// Vuelos Endpoints
	api.GET("/ciudades", handlers.GetAllCiudades)
	api.GET("/aviones", handlers.GetAllAviones)
	api.GET("/tiempos", handlers.GetTiempos)
	api.GET("/precios", handlers.GetPrecios)
	api.GET("/vuelos", handlers.GetAllVuelos)
	api.POST("/vuelos", handlers.CreateVuelo)
	api.GET("/vuelos/:vuelo_id/asientos", handlers.ListAsientos)
	api.PUT("/vuelos/:id/estado", handlers.UpdateEstadoVuelo)

	// Reservas / Asientos Endpoints
	api.POST("/reservas", handlers.ReservarAsiento)
	api.POST("/reservas/:id/cancelar", handlers.CancelarReserva)

	// Boletos Dashboard Endpoints
	api.GET("/boletos", handlers.ListBoletos)
	api.PATCH("/boletos/:id/estado", handlers.UpdateEstadoBoleto)

	// Sugerencias Dijkstra
	api.GET("/sugerencias/:criterio", handlers.GetSugerencias)
	
	// Agente Viajero (TSP) - Hamilton Path
	api.POST("/tsp", handlers.GetTSPRoute)
}
