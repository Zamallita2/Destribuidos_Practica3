package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"airres-api/data"
	"airres-api/db"
	"airres-api/models"
	"airres-api/services"
	"gorm.io/gorm"
)

// seedDemoFlights creates one future flight per real aircraft when the
// historical CSV has no future demo schedule. They are explicitly labelled.
func seedDemoFlights() {
	if os.Getenv("SEED_DEMO_FLIGHTS") == "false" || !db.IsAvailable(db.PGAmerica) || !db.IsAvailable(db.PGEuropaAsia) {
		return
	}
	matrixBytes, err := os.ReadFile(filepath.Join("data", "matrices.json"))
	if err != nil {
		log.Printf("[Demo] matrix unavailable: %v", err)
		return
	}
	var matrix data.MatricesJSON
	if err := json.Unmarshal(matrixBytes, &matrix); err != nil {
		log.Printf("[Demo] invalid matrix: %v", err)
		return
	}
	if len(matrix.Airports) == 0 {
		log.Printf("[Demo] matrix has no airports")
		return
	}
	var planes []models.Avion
	var cities []models.Ciudad
	var gates []models.Puerta
	db.PGAmerica.Order("id").Find(&planes)
	db.PGAmerica.Find(&cities)
	db.PGAmerica.Find(&gates)
	cityID := make(map[string]uint)
	cityCode := make(map[uint]string)
	gateID := make(map[uint]uint)
	for _, city := range cities {
		cityID[city.Codigo] = city.ID
		cityCode[city.ID] = city.Codigo
	}
	// A demo flight departs from where its aircraft last landed, after it lands.
	type lastPosition struct {
		IDAvion           uint
		IDDestino         uint
		LlegadaProgramada int64
	}
	lastByPlane := make(map[uint]lastPosition)
	for _, conn := range []*gorm.DB{db.PGAmerica, db.PGEuropaAsia} {
		var rows []lastPosition
		conn.Raw(`SELECT DISTINCT ON (id_avion) id_avion, id_destino, llegada_programada
			FROM vuelos WHERE id_estado_vuelo <> 7 ORDER BY id_avion, llegada_programada DESC`).Scan(&rows)
		for _, row := range rows {
			if row.LlegadaProgramada > lastByPlane[row.IDAvion].LlegadaProgramada {
				lastByPlane[row.IDAvion] = row
			}
		}
	}
	for _, gate := range gates {
		if gateID[gate.IDCiudad] == 0 {
			gateID[gate.IDCiudad] = gate.ID
		}
	}
	start := time.Now().UTC().Truncate(time.Hour).Add(24 * time.Hour)
	added := 0
	for index, plane := range planes {
		originCode := matrix.Airports[index%len(matrix.Airports)]
		departure := start.Add(time.Duration(index) * time.Hour).Unix()
		if last, ok := lastByPlane[plane.ID]; ok && cityCode[last.IDDestino] != "" {
			originCode = cityCode[last.IDDestino]
			if earliest := last.LlegadaProgramada + 3600; earliest > departure {
				departure = earliest
			}
		}
		originID := cityID[originCode]
		var existing int64
		owner, err := db.GetDBForOriginWrite(originID)
		if err != nil {
			continue
		}
		owner.Model(&models.Vuelo{}).Where("demo = ? AND id_avion = ? AND salida_programada > ?", true, plane.ID, time.Now().Unix()).Count(&existing)
		if existing > 0 {
			continue
		}
		var destinationCode string
		var hours float64
		for step := 1; step < len(matrix.Airports); step++ {
			candidate := matrix.Airports[(index+step)%len(matrix.Airports)]
			if allowed, _, _, duration := matrix.RouteAvailability(originCode, candidate); allowed {
				destinationCode, hours = candidate, duration
				break
			}
		}
		if destinationCode == "" {
			continue
		}
		flight := models.Vuelo{
			Demo: true, IDOrigen: originID, IDDestino: cityID[destinationCode],
			IDEstadoVuelo: 1, IDPuerta: gateID[originID], IDAvion: plane.ID,
			SalidaProgramada: departure, LlegadaProgramada: departure + int64(hours*3600),
			FechaSalida: departure, FechaLlegada: departure + int64(hours*3600),
			LamportClock: services.GlobalLamportClock.Tick(),
			VectorClock:  services.TickVectorClock(), SourceNode: services.NodeID,
		}
		err = owner.Transaction(func(tx *gorm.DB) error {
			namespace := "flight_eu"
			for _, city := range cities {
				if city.ID == originID && city.Region == "America" {
					namespace = "flight_am"
					break
				}
			}
			serial, err := db.NextDomainID(tx, namespace)
			if err != nil {
				return err
			}
			flight.ID = serial
			if err := tx.Create(&flight).Error; err != nil {
				return err
			}
			return services.QueueOutboxEvent(tx, "CREATE", "Vuelo", &flight)
		})
		if err != nil {
			log.Printf("[Demo] Could not create flight for aircraft %d: %v", plane.ID, err)
		} else {
			added++
		}
	}
	log.Printf("[Demo] Added %d future flights for booking demonstration", added)
}
