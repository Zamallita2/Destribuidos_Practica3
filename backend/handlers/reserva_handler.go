package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"airres-api/db"
	"airres-api/models"
	"airres-api/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListAsientos returns all seats for a specific flight with their occupancy status
func ListAsientos(c *gin.Context) {
	vueloIDStr := c.Param("vuelo_id")
	vueloID, _ := strconv.Atoi(vueloIDStr)

	tag := c.GetHeader("X-User-Country")
	dbConn, region := db.GetDBForCountry(tag)
	
	var vuelo models.Vuelo
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id": uint(vueloID)}).Decode(&vuelo)
	} else if dbConn != nil {
		if err := dbConn.First(&vuelo, vueloID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
			return
		}
	}

	if vuelo.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
		return
	}

	asientos := []models.Asiento{}
	boletos := []models.Boleto{}

	if region == "Asia" && db.MongoDatabase != nil {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		
		// Fetch seats
		collAsientos := db.MongoDatabase.Collection("asientos")
		opts := options.Find().SetSort(bson.M{"id": 1})
		cursorA, _ := collAsientos.Find(ctx, bson.M{"id_avion": vuelo.IDAvion}, opts)
		cursorA.All(ctx, &asientos)

		// Fetch boletos
		collBoletos := db.MongoDatabase.Collection("boletos")
		cursorB, _ := collBoletos.Find(ctx, bson.M{"id_vuelo": uint(vueloID), "estado": bson.M{"$ne": "ANNULLED"}})
		cursorB.All(ctx, &boletos)
	} else if dbConn != nil {
		dbConn.Order("id").Where("id_avion = ?", vuelo.IDAvion).Find(&asientos)
		dbConn.Where("id_vuelo = ? AND estado != ?", vueloID, "ANNULLED").Find(&boletos)
	}

	// Map seat ID to status and passenger info
	type SeatResponse struct {
		models.Asiento
		NombrePasajero string `json:"nombre_pasajero"`
		EmailPasajero  string `json:"email_pasajero"`
		Pasaporte      string `json:"pasaporte"`
	}

	response := []SeatResponse{}

	// Build boleto lookup map for this specific flight
	boletoByAsientoID := make(map[uint]models.Boleto)
	for _, b := range boletos {
		boletoByAsientoID[b.IDAsiento] = b
	}

	for _, s := range asientos {
		sr := SeatResponse{Asiento: s}
		if b, ok := boletoByAsientoID[s.ID]; ok {
			sr.Estado = b.Estado
			sr.NombrePasajero = b.NombrePasajero
			sr.EmailPasajero = b.EmailPasajero
			sr.Pasaporte = b.Pasaporte
		} else {
			sr.Estado = "AVAILABLE"
		}
		response = append(response, sr)
	}

	c.JSON(http.StatusOK, response)
}

// ReservarAsiento creates a Boleto and checks availability for the specific flight
func ReservarAsiento(c *gin.Context) {
	var payload struct {
		IDVuelo        uint    `json:"id_vuelo"`
		IDAsiento      uint    `json:"id_asiento"`
		NombrePasajero string  `json:"nombre_pasajero"`
		EmailPasajero  string  `json:"email_pasajero"`
		Pasaporte      string  `json:"pasaporte"`
		TiempoDeViaje  int     `json:"tiempo_de_viaje"`
		EstadoDeseado  string  `json:"estado"`
		Costo          float64 `json:"costo"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag := c.GetHeader("X-User-Country")
	dbConn, region := db.GetDBForCountry(tag)

	// Validate flight status: Cannot buy/reserve if flight state >= BOARDING (2)
	var vuelo models.Vuelo
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id": payload.IDVuelo}).Decode(&vuelo)
	} else if dbConn != nil {
		dbConn.First(&vuelo, payload.IDVuelo)
	}

	if vuelo.IDEstadoVuelo >= 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pueden comprar ni reservar boletos para un vuelo que ya está en proceso de abordaje, despegue o finalizado."})
		return
	}

	// Check if seat is already taken FOR THIS SPECIFIC FLIGHT
	var existingBoleto models.Boleto
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{
			"id_vuelo":   payload.IDVuelo,
			"id_asiento": payload.IDAsiento,
			"estado":     bson.M{"$ne": "ANNULLED"},
		}).Decode(&existingBoleto)
	} else if dbConn != nil {
		dbConn.Where("id_vuelo = ? AND id_asiento = ? AND estado != ?", payload.IDVuelo, payload.IDAsiento, "ANNULLED").First(&existingBoleto)
	}

	if existingBoleto.IDBoleto != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "El asiento ya está reservado o vendido para este vuelo."})
		return
	}

	// Create Boleto
	nuevoBoleto := models.Boleto{
		NombrePasajero: payload.NombrePasajero,
		EmailPasajero:  payload.EmailPasajero,
		Pasaporte:      payload.Pasaporte,
		TiempoDeViaje:  payload.TiempoDeViaje,
		IDVuelo:        payload.IDVuelo,
		IDAsiento:      payload.IDAsiento,
		Costo:          payload.Costo,
		Estado:         payload.EstadoDeseado,
	}

	nuevoBoleto.LamportClock = services.GlobalLamportClock.Tick()
	
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if nuevoBoleto.IDBoleto == 0 {
			count, _ := coll.CountDocuments(ctx, bson.M{})
			nuevoBoleto.IDBoleto = uint(count + 1)
		}
		coll.InsertOne(ctx, nuevoBoleto)
	} else if dbConn != nil {
		dbConn.Create(&nuevoBoleto)
	}
	go services.SendSyncEvent("CREATE", "Boleto", &nuevoBoleto, nuevoBoleto.LamportClock)

	c.JSON(http.StatusOK, nuevoBoleto)
}

// CancelarReserva changes seat state back to AVAILABLE processing the "15 mins" logistics logic
func CancelarReserva(c *gin.Context) {
	boletoIDStr := c.Param("id")
	boletoID, _ := strconv.Atoi(boletoIDStr)

	tag := c.GetHeader("X-User-Country")
	dbConn, region := db.GetDBForCountry(tag)

	var boleto models.Boleto
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id_boleto": uint(boletoID)}).Decode(&boleto)
	} else if dbConn != nil {
		if err := dbConn.First(&boleto, boletoID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
			return
		}
	}

	if boleto.IDBoleto == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
		return
	}

	// Validate flight status for cancellation: Cannot cancel if flight state >= BOARDING (2)
	var vuelo models.Vuelo
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id": boleto.IDVuelo}).Decode(&vuelo)
	} else if dbConn != nil {
		dbConn.First(&vuelo, boleto.IDVuelo)
	}

	if vuelo.IDEstadoVuelo >= 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se puede cancelar una reserva cuando el vuelo ya está en abordaje, despegue o finalizado."})
		return
	}

	// Cambiar Boleto a Anulado
	boleto.Estado = "ANNULLED"
	boleto.LamportClock = services.GlobalLamportClock.Tick()

	if region == "Asia" && db.MongoDatabase != nil {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		db.MongoDatabase.Collection("boletos").ReplaceOne(ctx, bson.M{"id_boleto": boleto.IDBoleto}, boleto)
	} else if dbConn != nil {
		dbConn.Save(&boleto)
	}

	go services.SendSyncEvent("UPDATE", "Boleto", &boleto, boleto.LamportClock)

	c.JSON(http.StatusOK, gin.H{"message": "Reserva cancelada. El asiento procesará disponibilidad en los ecosistemas (retraso 15s) debido a política de reembolso."})
}

// ListBoletos returns all tickets generated in the current region
func ListBoletos(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	_, region := db.GetDBForCountry(tag)
	
	boletos := []models.Boleto{}
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cursor, _ := coll.Find(ctx, bson.M{})
		cursor.All(ctx, &boletos)
	} else {
		dbConn, _ := db.GetDBForCountry(tag)
		if dbConn != nil {
			dbConn.Find(&boletos)
		}
	}
	c.JSON(http.StatusOK, boletos)
}

// UpdateEstadoBoleto changes the state of a ticket (dashboard management)
func UpdateEstadoBoleto(c *gin.Context) {
	boletoIDStr := c.Param("id")
	boletoID, _ := strconv.Atoi(boletoIDStr)

	var payload struct {
		Estado string `json:"estado"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validStates := map[string]bool{"RESERVED": true, "SALED": true, "ANNULLED": true}
	if !validStates[payload.Estado] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado inválido. Los estados permitidos son: RESERVED, SALED, ANNULLED"})
		return
	}

	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)

	var boleto models.Boleto
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id_boleto": uint(boletoID)}).Decode(&boleto)
	} else if dbConn != nil {
		if err := dbConn.First(&boleto, boletoID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
			return
		}
	}

	if boleto.IDBoleto == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
		return
	}

	// Validate flight status for ticket state change
	var vuelo models.Vuelo
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id": boleto.IDVuelo}).Decode(&vuelo)
	} else if dbConn != nil {
		dbConn.First(&vuelo, boleto.IDVuelo)
	}

	if vuelo.IDEstadoVuelo >= 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se puede modificar o anular el boleto porque el vuelo ya está en abordaje, despegue o finalizado."})
		return
	}

	boleto.Estado = payload.Estado
	boleto.LamportClock = services.GlobalLamportClock.Tick()

	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.ReplaceOne(ctx, bson.M{"id_boleto": boleto.IDBoleto}, boleto)
	} else if dbConn != nil {
		dbConn.Save(&boleto)
	}

	go services.SendSyncEvent("UPDATE", "Boleto", &boleto, boleto.LamportClock)
	c.JSON(http.StatusOK, boleto)
}
