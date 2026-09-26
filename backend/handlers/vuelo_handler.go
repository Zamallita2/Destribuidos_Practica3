package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"airres-api/db"
	"airres-api/models"
	"airres-api/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var flightCreationMu sync.Mutex

// GetAllVuelos keeps the legacy array response for booking screens. The
// catalog view adds a total so the management screen can page through the
// entire imported dataset without downloading thousands of flights at once.
func GetAllVuelos(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)
	c.Header("X-Data-Source", region)

	vuelos := []models.Vuelo{}
	limit := 100
	offset := 0
	if requested, err := strconv.Atoi(c.Query("limit")); err == nil && requested > 0 && requested <= 500 {
		limit = requested
	}
	if requested, err := strconv.Atoi(c.Query("offset")); err == nil && requested >= 0 {
		offset = requested
	}
	all := c.Query("scope") == "all"
	pageView := c.Query("view") == "page"
	var flightID, originID, destinationID uint64
	for _, filter := range []struct {
		name  string
		value *uint64
	}{
		{"id", &flightID}, {"origin", &originID}, {"destination", &destinationID},
	} {
		if raw := c.Query(filter.name); raw != "" {
			parsed, err := strconv.ParseUint(raw, 10, 32)
			if err != nil || parsed == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Filtro de vuelo inválido: " + filter.name})
				return
			}
			*filter.value = parsed
		}
	}
	var total int64
	var importedCount, demoCount int64

	if region == "Mongo" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		findOptions := options.Find()
		findOptions.SetLimit(int64(limit)).SetSkip(int64(offset))
		filter := bson.M{}
		if flightID > 0 {
			filter["id"] = flightID
		}
		if originID > 0 {
			filter["id_origen"] = originID
		}
		if destinationID > 0 {
			filter["id_destino"] = destinationID
		}
		if all {
			findOptions.SetSort(bson.D{{Key: "salida_programada", Value: -1}, {Key: "id", Value: -1}})
		} else {
			filter["salida_programada"] = bson.M{"$gte": time.Now().Unix()}
			findOptions.SetSort(bson.D{{Key: "salida_programada", Value: 1}, {Key: "id", Value: 1}})
		}
		if pageView {
			count, err := coll.CountDocuments(ctx, filter)
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
			total = count
			demoCount, err = coll.CountDocuments(ctx, bson.M{"demo": true})
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
			importedCount, err = coll.CountDocuments(ctx, bson.M{"demo": bson.M{"$ne": true}})
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
		}
		cursor, err := coll.Find(ctx, filter, findOptions)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)
		if err := cursor.All(ctx, &vuelos); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
	} else if dbConn != nil {
		query := dbConn.Model(&models.Vuelo{})
		if flightID > 0 {
			query = query.Where("id = ?", flightID)
		}
		if originID > 0 {
			query = query.Where("id_origen = ?", originID)
		}
		if destinationID > 0 {
			query = query.Where("id_destino = ?", destinationID)
		}
		if all {
			query = query.Order("salida_programada DESC, id DESC")
		} else {
			query = query.Where("salida_programada >= ?", time.Now().Unix()).Order("salida_programada ASC, id ASC")
		}
		if pageView {
			if err := query.Count(&total).Error; err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
			if err := dbConn.Raw("SELECT COUNT(*) FILTER (WHERE demo IS TRUE), COUNT(*) FILTER (WHERE demo IS NOT TRUE) FROM vuelos").Row().Scan(&demoCount, &importedCount); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
		}
		if err := query.Limit(limit).Offset(offset).Find(&vuelos).Error; err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No database connection for region"})
		return
	}

	if pageView {
		c.JSON(http.StatusOK, gin.H{"items": vuelos, "total": total, "limit": limit, "offset": offset,
			"catalog": gin.H{"imported": importedCount, "demo": demoCount}})
		return
	}
	c.JSON(http.StatusOK, vuelos)
}

// GetAllCiudades returns all available airports/cities
func GetAllCiudades(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)

	ciudades := []models.Ciudad{}
	if region == "Mongo" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("ciudades")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if cursor, err := coll.Find(ctx, bson.M{}); err == nil {
			cursor.All(ctx, &ciudades)
		}
	} else if dbConn != nil {
		dbConn.Find(&ciudades)
	}

	c.JSON(http.StatusOK, ciudades)
}

// GetAllAviones returns all available aircraft
func GetAllAviones(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)

	aviones := []models.Avion{}
	if region == "Mongo" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("aviones")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if cursor, err := coll.Find(ctx, bson.M{}); err == nil {
			cursor.All(ctx, &aviones)
		}
	} else if dbConn != nil {
		dbConn.Find(&aviones)
	}

	c.JSON(http.StatusOK, aviones)
}

// GetTiempos returns the travel time matrix from the database
func GetTiempos(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)

	var detalles models.DetallesVuelos
	if region == "Mongo" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("detalles_vuelos")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var result map[string]interface{}
		if err := coll.FindOne(ctx, bson.M{}).Decode(&result); err == nil {
			if val, ok := result["matriz_tiempos"]; ok {
				jsonBytes, _ := json.Marshal(val)
				_ = json.Unmarshal(jsonBytes, &detalles.MatrizTiempos)
			}
		}
	} else if dbConn != nil {
		if err := dbConn.First(&detalles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Travel matrix not found in DB"})
			return
		}
	}

	c.Data(http.StatusOK, "application/json", detalles.MatrizTiempos)
}

// GetPrecios returns the price matrices from the database
func GetPrecios(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)

	var precios models.Precios
	if region == "Mongo" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("precios")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var result map[string]interface{}
		if err := coll.FindOne(ctx, bson.M{}).Decode(&result); err == nil {
			if valR, ok := result["matriz_precios_regular"]; ok {
				jsonBytes, _ := json.Marshal(valR)
				_ = json.Unmarshal(jsonBytes, &precios.MatrizPreciosRegular)
			}
			if valV, ok := result["matriz_precios_vip"]; ok {
				jsonBytes, _ := json.Marshal(valV)
				_ = json.Unmarshal(jsonBytes, &precios.MatrizPreciosVip)
			}
		}
	} else if dbConn != nil {
		if err := dbConn.First(&precios).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Price matrix not found in DB"})
			return
		}
	}

	c.JSON(http.StatusOK, precios)
}

// CreateVuelo inserts a new flight into DB
func CreateVuelo(c *gin.Context) {
	var nVuelo models.Vuelo
	if err := c.ShouldBindJSON(&nVuelo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbConn, err := db.GetDBForOriginWrite(nVuelo.IDOrigen)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aeropuerto de origen inválido"})
		return
	}
	if nVuelo.SalidaProgramada <= time.Now().Unix() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El vuelo nuevo debe salir en el futuro"})
		return
	}
	flightCreationMu.Lock()
	defer flightCreationMu.Unlock()
	err = dbConn.Transaction(func(tx *gorm.DB) error {
		var plane models.Avion
		if err := tx.First(&plane, nVuelo.IDAvion).Error; err != nil {
			return err
		}
		duration, err := services.DurationForFlight(tx, &nVuelo)
		if err != nil {
			return err
		}
		_, economyErr := services.FareForFlight(tx, &nVuelo, "REGULAR")
		_, firstErr := services.FareForFlight(tx, &nVuelo, "VIP")
		if economyErr != nil && firstErr != nil {
			return economyErr
		}
		var gate models.Puerta
		if nVuelo.IDPuerta == 0 || tx.Where("id = ? AND id_ciudad = ?", nVuelo.IDPuerta, nVuelo.IDOrigen).First(&gate).Error != nil {
			if err := tx.Where("id_ciudad = ?", nVuelo.IDOrigen).First(&gate).Error; err != nil {
				return err
			}
			nVuelo.IDPuerta = gate.ID
		}
		nVuelo.LlegadaProgramada = nVuelo.SalidaProgramada + duration
		for _, conn := range []*gorm.DB{db.PGAmerica, db.PGEuropaAsia} {
			if !db.IsAvailable(conn) {
				return fmt.Errorf("no se puede comprobar la disponibilidad del avión en todos los nodos")
			}
			var count int64
			if err := conn.Model(&models.Vuelo{}).Where("id_avion = ? AND id_estado_vuelo <> ? AND salida_programada < ? AND llegada_programada > ?", nVuelo.IDAvion, 7, nVuelo.LlegadaProgramada, nVuelo.SalidaProgramada).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("el avión ya tiene un vuelo durante ese horario")
			}
		}
		nVuelo.FechaSalida = nVuelo.SalidaProgramada
		nVuelo.FechaLlegada = nVuelo.LlegadaProgramada
		nVuelo.IDEstadoVuelo = 1
		nVuelo.LamportClock = services.GlobalLamportClock.Tick()
		nVuelo.VectorClock = services.TickVectorClock()
		nVuelo.SourceNode = services.NodeID
		var origin models.Ciudad
		if err := tx.First(&origin, nVuelo.IDOrigen).Error; err != nil {
			return err
		}
		namespace := "flight_eu"
		if origin.Region == "America" {
			namespace = "flight_am"
		}
		serial, err := db.NextDomainID(tx, namespace)
		if err != nil {
			return err
		}
		nVuelo.ID = serial
		if err := tx.Create(&nVuelo).Error; err != nil {
			return err
		}
		return services.QueueOutboxEvent(tx, "CREATE", "Vuelo", &nVuelo)
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nVuelo)
}

func GetAllPuertas(c *gin.Context) {
	var gates []models.Puerta
	conn := db.PGAmerica
	if !db.IsAvailable(conn) {
		conn = db.PGEuropaAsia
	}
	if !db.IsAvailable(conn) || conn.Find(&gates).Error != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Puertas no disponibles"})
		return
	}
	c.JSON(http.StatusOK, gates)
}

func GetVuelo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	owner := db.PGAmerica
	if id >= 1000000000 {
		owner = db.PGEuropaAsia
	}
	if !db.IsAvailable(owner) && db.IsMongoAvailable() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var snapshot models.Vuelo
		if err := db.MongoDatabase.Collection("vuelos").FindOne(ctx, bson.M{"id": uint(id)}).Decode(&snapshot); err == nil {
			c.Header("X-Data-Source", "Mongo")
			c.JSON(http.StatusOK, snapshot)
			return
		}
	}
	conn, flight, err := db.GetDBForFlightWrite(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
		return
	}
	if conn == db.PGAmerica {
		c.Header("X-Data-Source", "America")
	} else {
		c.Header("X-Data-Source", "Europa")
	}
	c.JSON(http.StatusOK, flight)
}

// UpdateEstadoVuelo transitions the flight state manually
func UpdateEstadoVuelo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	idEstado, err := strconv.ParseUint(c.Query("id_estado"), 10, 8)
	if err != nil || idEstado < 1 || idEstado > 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado inválido"})
		return
	}
	dbConn, _, err := db.GetDBForFlightWrite(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
		return
	}
	var vuelo models.Vuelo
	err = dbConn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&vuelo, id).Error; err != nil {
			return err
		}
		vuelo.IDEstadoVuelo = uint(idEstado)
		if vuelo.SalidaProgramada > time.Now().Unix() {
			vuelo.IDEstadoVuelo = 1
		}
		services.ObserveClock(vuelo.LamportClock, vuelo.VectorClock)
		vuelo.LamportClock = services.GlobalLamportClock.Tick()
		vuelo.VectorClock = services.TickVectorClock()
		vuelo.SourceNode = services.NodeID
		if err := tx.Save(&vuelo).Error; err != nil {
			return err
		}
		return services.QueueOutboxEvent(tx, "UPDATE", "Vuelo", &vuelo)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vuelo)
}
