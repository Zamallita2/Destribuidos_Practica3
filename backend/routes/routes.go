package routes

import (
	"airres-api/handlers"
	"github.com/gin-gonic/gin"
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
			"status":  "success",
			"country": country,
			"message": "Connected to cluster",
		})
	})

	// Vuelos Endpoints
	api.GET("/ciudades", handlers.GetAllCiudades)
	api.GET("/aviones", handlers.GetAllAviones)
	api.GET("/puertas", handlers.GetAllPuertas)
	api.GET("/tiempos", handlers.GetTiempos)
	api.GET("/precios", handlers.GetPrecios)
	api.GET("/vuelos", handlers.GetAllVuelos)
	api.GET("/vuelos/:id", handlers.GetVuelo)
	api.GET("/diagnostico/continuidad", handlers.GetContinuityReport)
	api.GET("/dashboard", handlers.GetDashboardSummary)
	api.GET("/dashboard/vuelos/:id", handlers.GetFlightDashboard)
	api.POST("/vuelos", handlers.CreateVuelo)
	api.GET("/vuelos/:id/asientos", handlers.ListAsientos)
	api.PUT("/vuelos/:id/estado", handlers.UpdateEstadoVuelo)

	// Reservas / Asientos Endpoints
	api.POST("/reservas", handlers.ReservarAsiento)
	api.POST("/reservas/:id/cancelar", handlers.CancelarReserva)

	// Boletos Dashboard Endpoints
	api.GET("/boletos", handlers.ListBoletos)
	api.GET("/boletos/:id/pase", handlers.GetBoardingPass)
	api.GET("/boletos/:id/qr.png", handlers.GetBoardingQRCode)
	api.GET("/boletos/:id/wallet/qr.png", handlers.GetWalletDownloadQRCode)
	api.GET("/boletos/:id/validar", handlers.ValidateBoardingPass)
	api.GET("/wallet/capabilities", handlers.WalletCapabilities)
	api.GET("/network/qr-address", handlers.GetQRNetworkAddress)
	api.GET("/boletos/:id/wallet/demo.pkpass", handlers.GetDemoWalletPass)
	api.GET("/boletos/:id/wallet/demo.zip", handlers.GetDemoWalletPassZip)
	api.GET("/boletos/:id/wallet/google", handlers.GetGoogleWalletLink)
	api.GET("/boletos/:id/wallet/apple.pkpass", handlers.GetAppleWalletPass)
	api.PATCH("/boletos/:id/estado", handlers.UpdateEstadoBoleto)

	// Sugerencias Dijkstra
	api.GET("/sugerencias/:criterio", handlers.GetSugerencias)

	// Agente Viajero (TSP) - Hamilton Path
	api.POST("/tsp", handlers.GetTSPRoute)
}
