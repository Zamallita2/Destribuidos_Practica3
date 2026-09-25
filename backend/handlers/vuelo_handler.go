package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"airres-api/db"
	"airres-api/models"
	"airres-api/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetAllVuelos returns all flights with their status
func GetAllVuelos(c *gin.Context) {
	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)
	
	vuelos := []models.Vuelo{}

	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		findOptions := options.Find()
		findOptions.SetLimit(100)
		cursor, _ := coll.Find(ctx, bson.M{}, findOptions)
		cursor.All(ctx, &vuelos)
	} else if dbConn != nil {
		dbConn.Limit(100).Find(&vuelos)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No database connection for region"})
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
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("ciudades")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cursor, _ := coll.Find(ctx, bson.M{})
		cursor.All(ctx, &ciudades)
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
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("aviones")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		cursor, _ := coll.Find(ctx, bson.M{})
		cursor.All(ctx, &aviones)
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
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("detalles_vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
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
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("precios")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
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

	tag := c.GetHeader("X-Region")
	if tag == "" {
		tag = c.GetHeader("X-User-Country")
	}
	dbConn, region := db.GetDBForCountry(tag)
	
	nVuelo.LamportClock = services.GlobalLamportClock.Tick()
	nVuelo.VectorClock = services.TickVectorClock()

	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		// Basic ID logic for demo
		if nVuelo.ID == 0 {
			count, _ := coll.CountDocuments(ctx, bson.M{})
			nVuelo.ID = uint(count + 1)
		}
		coll.InsertOne(ctx, nVuelo)
	} else if dbConn != nil {
		if err := dbConn.Create(&nVuelo).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar el vuelo"})
			return
		}
	}

	// Disparar sincronización
	go services.SendSyncEvent("CREATE", "Vuelo", &nVuelo, nVuelo.LamportClock, nVuelo.VectorClock)

	c.JSON(http.StatusOK, nVuelo)
}

// UpdateEstadoVuelo transitions the flight state manually
func UpdateEstadoVuelo(c *gin.Context) {
	idStr := c.Param("id")
	nuevoEstadoIdStr := c.Query("id_estado")

	id, _ := strconv.Atoi(idStr)
	idEstado, _ := strconv.Atoi(nuevoEstadoIdStr)

	tag := c.GetHeader("X-User-Country")
	dbConn, region := db.GetDBForCountry(tag)

	var vuelo models.Vuelo
	if region == "Asia" && db.MongoDatabase != nil {
		coll := db.MongoDatabase.Collection("vuelos")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		coll.FindOne(ctx, bson.M{"id": uint(id)}).Decode(&vuelo)
		vuelo.IDEstadoVuelo = uint(idEstado)
		vuelo.LamportClock = services.GlobalLamportClock.Tick()
		vuelo.VectorClock = services.TickVectorClock()
		coll.ReplaceOne(ctx, bson.M{"id": uint(id)}, vuelo)
	} else if dbConn != nil {
		if err := dbConn.First(&vuelo, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Vuelo no encontrado"})
			return
		}
		vuelo.IDEstadoVuelo = uint(idEstado)
		vuelo.LamportClock = services.GlobalLamportClock.Tick()
		vuelo.VectorClock = services.TickVectorClock()
		dbConn.Save(&vuelo)
	}

	go services.SendSyncEvent("UPDATE", "Vuelo", &vuelo, vuelo.LamportClock, vuelo.VectorClock)
	c.JSON(http.StatusOK, vuelo)
}
