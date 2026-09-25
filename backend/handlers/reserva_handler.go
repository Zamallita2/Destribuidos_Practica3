package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"airres-api/db"
	"airres-api/models"
	"airres-api/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListAsientos returns all seats for a specific flight with their occupancy status
func ListAsientos(c *gin.Context) {
	vueloID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	owner, vuelo, err := db.GetDBForFlightWrite(uint(vueloID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
		return
	}
	var manifest models.OcupacionVuelo
	if err := owner.Transaction(func(tx *gorm.DB) error {
		var ensureErr error
		manifest, ensureErr = services.EnsureFlightOccupancy(tx, vuelo)
		return ensureErr
	}); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	assignments, err := services.DecodeManifest(manifest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	asientos := []models.Asiento{}
	boletos := []models.Boleto{}
	owner.Order("id").Where("id_avion = ?", vuelo.IDAvion).Find(&asientos)
	owner.Where("id_vuelo = ? AND estado IN ?", vueloID, []string{"SALED", "RESERVED", "REFUNDED"}).Find(&boletos)
	_, regularErr := services.FareForFlight(owner, &vuelo, "REGULAR")
	_, vipErr := services.FareForFlight(owner, &vuelo, "VIP")

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
	manifestBySeat := make(map[uint]models.FlightSeatAssignment, len(assignments))
	for _, assignment := range assignments {
		manifestBySeat[assignment.SeatID] = assignment
	}

	for _, s := range asientos {
		sr := SeatResponse{Asiento: s}
		sr.Estado = "AVAILABLE"
		if (s.Clase == "REGULAR" && regularErr != nil) || (s.Clase == "VIP" && vipErr != nil) {
			sr.Estado = "BLOCKED"
		}
		if assigned, ok := manifestBySeat[s.ID]; ok {
			sr.Estado = assigned.Status
			sr.NombrePasajero = assigned.PassengerName
			sr.EmailPasajero = assigned.PassengerEmail
			sr.Pasaporte = assigned.Passport
		}
		if b, ok := boletoByAsientoID[s.ID]; ok {
			sr.Estado = b.Estado
			sr.NombrePasajero = b.NombrePasajero
			sr.EmailPasajero = b.EmailPasajero
			sr.Pasaporte = b.Pasaporte
		}
		response = append(response, sr)
	}

	c.JSON(http.StatusOK, response)
}

// ReservarAsiento creates a Boleto and checks availability for the specific flight
func ReservarAsiento(c *gin.Context) {
	var payload struct {
		IDVuelo          uint    `json:"id_vuelo"`
		IDAsiento        uint    `json:"id_asiento"`
		NombrePasajero   string  `json:"nombre_pasajero"`
		EmailPasajero    string  `json:"email_pasajero"`
		Pasaporte        string  `json:"pasaporte"`
		PurchaseTimeZone string  `json:"time_zone_compra"`
		TiempoDeViaje    int     `json:"tiempo_de_viaje"`
		EstadoDeseado    string  `json:"estado"`
		Costo            float64 `json:"costo"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.PurchaseTimeZone == "" {
		payload.PurchaseTimeZone = "UTC"
	}
	if _, err := time.LoadLocation(payload.PurchaseTimeZone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Zona horaria de compra inválida"})
		return
	}

	dbConn, _, err := db.GetDBForFlightWrite(payload.IDVuelo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado o nodo propietario no disponible"})
		return
	}
	var nuevoBoleto models.Boleto
	err = dbConn.Transaction(func(tx *gorm.DB) error {
		var vuelo models.Vuelo
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&vuelo, payload.IDVuelo).Error; err != nil {
			return err
		}
		var asiento models.Asiento
		if err := tx.First(&asiento, payload.IDAsiento).Error; err != nil {
			return errors.New("asiento_no_encontrado")
		}
		if err := services.ValidateBooking(vuelo, asiento, payload.EstadoDeseado, time.Now()); err != nil {
			return err
		}
		manifest, err := services.EnsureFlightOccupancy(tx, vuelo)
		if err != nil {
			return err
		}
		assignments, err := services.DecodeManifest(manifest)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if assignment.SeatID == payload.IDAsiento {
				return errors.New("asiento_ocupado")
			}
		}
		var existing models.Boleto
		if err := tx.Where("id_vuelo = ? AND id_asiento = ? AND estado IN ?", payload.IDVuelo, payload.IDAsiento,
			[]string{"RESERVED", "SALED", "REFUNDED"}).First(&existing).Error; err == nil {
			return errors.New("asiento_ocupado")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		fare, err := services.FareForFlight(tx, &vuelo, asiento.Clase)
		if err != nil {
			return err
		}
		nuevoBoleto = models.Boleto{
			NombrePasajero:   payload.NombrePasajero,
			EmailPasajero:    payload.EmailPasajero,
			Pasaporte:        payload.Pasaporte,
			PurchaseTimeZone: payload.PurchaseTimeZone,
			TiempoDeViaje:    int((vuelo.LlegadaProgramada - vuelo.SalidaProgramada) / 3600),
			IDVuelo:          payload.IDVuelo, IDAsiento: payload.IDAsiento,
			Clase: asiento.Clase, Costo: fare, Estado: payload.EstadoDeseado,
			LamportClock: services.GlobalLamportClock.Tick(),
			VectorClock:  services.TickVectorClock(),
			SourceNode:   services.NodeID,
		}
		namespace := "ticket_am"
		if vuelo.ID >= 1000000000 {
			namespace = "ticket_eu"
		}
		serial, err := db.NextDomainID(tx, namespace)
		if err != nil {
			return err
		}
		nuevoBoleto.IDBoleto = serial
		if err := tx.Create(&nuevoBoleto).Error; err != nil {
			return err
		}
		return services.QueueOutboxEvent(tx, "CREATE", "Boleto", &nuevoBoleto)
	})
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "asiento_ocupado" || strings.Contains(err.Error(), "unique_active_seat") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if err := services.ConfirmTicketReplication(dbConn, nuevoBoleto); err != nil {
		c.JSON(http.StatusAccepted, gin.H{"id_boleto": nuevoBoleto.IDBoleto, "estado": nuevoBoleto.Estado, "replication_pending": true, "message": "Boleto guardado; confirmación de réplica pendiente"})
		return
	}
	c.JSON(http.StatusOK, nuevoBoleto)
}

// CancelarReserva starts the configurable refund window.
func CancelarReserva(c *gin.Context) {
	boletoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de boleto inválido"})
		return
	}
	dbConn, _, err := db.GetDBForTicketWrite(uint(boletoID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
		return
	}
	var boleto models.Boleto
	err = dbConn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&boleto, boletoID).Error; err != nil {
			return err
		}
		var vuelo models.Vuelo
		if err := tx.First(&vuelo, boleto.IDVuelo).Error; err != nil {
			return err
		}
		if (vuelo.IDEstadoVuelo != 1 && vuelo.IDEstadoVuelo != 8) || vuelo.SalidaProgramada <= time.Now().Unix() {
			return errors.New("vuelo_cerrado")
		}
		if boleto.Estado != "RESERVED" && boleto.Estado != "SALED" {
			return errors.New("boleto_no_cancelable")
		}
		boleto.Estado = "REFUNDED"
		boleto.AvailableAt = services.RefundAvailableAt(time.Now(), services.ConfiguredRefundDelay())
		boleto.LamportClock = services.GlobalLamportClock.Tick()
		boleto.VectorClock = services.TickVectorClock()
		boleto.SourceNode = services.NodeID
		if err := tx.Save(&boleto).Error; err != nil {
			return err
		}
		return services.QueueOutboxEvent(tx, "UPDATE", "Boleto", &boleto)
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Devolución iniciada", "available_at": boleto.AvailableAt})
}

// ListBoletos returns all tickets generated in the current region
func ListBoletos(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)
	c.Header("X-Data-Source", region)

	boletos := []models.Boleto{}
	if region == "Mongo" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("boletos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cursor, _ := coll.Find(ctx, bson.M{})
		cursor.All(ctx, &boletos)
	} else if dbConn != nil {
		dbConn.Find(&boletos)
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Ningún servidor disponible para consultar boletos"})
		return
	}
	c.JSON(http.StatusOK, boletos)
}

// UpdateEstadoBoleto changes the state of a ticket (dashboard management)
func UpdateEstadoBoleto(c *gin.Context) {
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
	if payload.Estado == "ANNULLED" {
		CancelarReserva(c)
		return
	}
	boletoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de boleto inválido"})
		return
	}
	dbConn, _, err := db.GetDBForTicketWrite(uint(boletoID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Boleto no encontrado"})
		return
	}
	var boleto models.Boleto
	err = dbConn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&boleto, boletoID).Error; err != nil {
			return err
		}
		var vuelo models.Vuelo
		if err := tx.First(&vuelo, boleto.IDVuelo).Error; err != nil {
			return err
		}
		if (vuelo.IDEstadoVuelo != 1 && vuelo.IDEstadoVuelo != 8) || vuelo.SalidaProgramada <= time.Now().Unix() {
			return errors.New("vuelo_cerrado")
		}
		if boleto.Estado != "RESERVED" || payload.Estado != "SALED" {
			return errors.New("transicion_invalida")
		}
		boleto.Estado = "SALED"
		boleto.LamportClock = services.GlobalLamportClock.Tick()
		boleto.VectorClock = services.TickVectorClock()
		boleto.SourceNode = services.NodeID
		if err := tx.Save(&boleto).Error; err != nil {
			return err
		}
		return services.QueueOutboxEvent(tx, "UPDATE", "Boleto", &boleto)
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, boleto)
}
