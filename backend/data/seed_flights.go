package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"airres-api/models"

	"gorm.io/gorm"
)

// statusMap maps CSV statuses to DB estado_vuelo IDs
var statusMap = map[string]uint{
	"SCHEDULED": 1,
	"BOARDING":  2,
	"DEPARTED":  3,
	"IN_FLIGHT": 4,
	"LANDED":    5,
	"ARRIVED":   6,
	"CANCELLED": 7,
	"DELAYED":   8,
}

// MatricesJSON maps to matrices.json to check supported routes
type MatricesJSON struct {
	Airports        []string                       `json:"airports"`
	TravelTime      map[string]map[string]float64  `json:"travel_time"`
	EconomyFares    map[string]map[string]*float64 `json:"economy_fares"`
	FirstClassFares map[string]map[string]*float64 `json:"first_class_fares"`
}

// RouteAvailability checks the selected direction. A route may sell either
// class independently, but it must have a positive flight duration.
func (m MatricesJSON) RouteAvailability(origin, destination string) (allowed, economy, vip bool, hours float64) {
	if origin == destination {
		return false, false, false, 0
	}
	hours = m.TravelTime[origin][destination]
	if hours <= 0 {
		return false, false, false, 0
	}
	ecoFare := m.EconomyFares[origin][destination]
	firstFare := m.FirstClassFares[origin][destination]
	economy = ecoFare != nil && *ecoFare > 0
	vip = firstFare != nil && *firstFare > 0
	if !economy && !vip {
		return false, false, false, 0
	}
	return true, economy, vip, hours
}

// SeedFlightsFromCSV adds missing supported rows without changing existing IDs.
func SeedFlightsFromCSV(dbConn *gorm.DB, csvPath, ownerRegion string) {
	if _, err := ImportFlightsFromCSV(dbConn, csvPath, filepath.Join("data", "matrices.json"), ownerRegion); err != nil {
		log.Printf("[Seed Flights] Import failed: %v", err)
	}
}

// ImportFlightsFromCSV imports one region and reports write failures to the caller.
func ImportFlightsFromCSV(dbConn *gorm.DB, csvPath, matrixPath, ownerRegion string) (int, error) {
	if dbConn == nil {
		return 0, fmt.Errorf("nodo %s no disponible", ownerRegion)
	}

	// Open CSV
	f, err := os.Open(csvPath)
	if err != nil {
		return 0, fmt.Errorf("abrir CSV: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	// Load matrices.json to validate routes (Rule 3)
	var matrices MatricesJSON
	matrixBytes, err := os.ReadFile(matrixPath)
	if err == nil {
		if err := json.Unmarshal(matrixBytes, &matrices); err != nil {
			return 0, fmt.Errorf("matrices inválidas: %w", err)
		}
	} else {
		return 0, fmt.Errorf("leer matrices: %w", err)
	}

	// Read header
	if _, err := reader.Read(); err != nil {
		return 0, fmt.Errorf("encabezado CSV: %w", err)
	}

	// Create a rejection log file
	logFile, err := os.Create(filepath.Join("data", "rejected_flights.log"))
	if err == nil {
		defer logFile.Close()
		logFile.WriteString("=== Registro de Vuelos Rechazados ===\n\n")
	} else {
		logFile = nil
		log.Printf("[Seed Flights] Warning: could not create rejection log: %v", err)
	}

	// Helper function to write to log
	logRejection := func(row []string, reason string) {
		if logFile != nil {
			flightInfo := strings.Join(row, ", ")
			logFile.WriteString(fmt.Sprintf("RECHAZADO: %s | Motivo: %s\n", flightInfo, reason))
		}
	}

	// Build ciudad lookup: codigo -> id
	var ciudades []models.Ciudad
	dbConn.Find(&ciudades)
	ciudadByCode := make(map[string]uint)
	for _, c := range ciudades {
		ciudadByCode[c.Codigo] = c.ID
	}
	var planes []models.Avion
	dbConn.Select("id").Find(&planes)
	validPlane := make(map[uint]bool, len(planes))
	for _, plane := range planes {
		validPlane[plane.ID] = true
	}

	// Build puerta lookup: puerta_str -> id
	var puertas []models.Puerta
	dbConn.Find(&puertas)
	puertaByName := make(map[uint]map[string]uint)
	fallbackGate := make(map[uint]uint)
	for _, p := range puertas {
		if puertaByName[p.IDCiudad] == nil {
			puertaByName[p.IDCiudad] = make(map[string]uint)
		}
		puertaByName[p.IDCiudad][p.Puerta] = p.ID
		if fallbackGate[p.IDCiudad] == 0 {
			fallbackGate[p.IDCiudad] = p.ID
		}
	}
	if len(puertas) == 0 {
		return 0, fmt.Errorf("no hay puertas disponibles")
	}

	// Existing data may have been imported under the old both-fares rule.
	// Count matching rows so even duplicate CSV records are treated correctly.
	var existing []models.Vuelo
	if err := dbConn.Select("id_avion", "id_origen", "id_destino", "salida_programada").Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("consultar vuelos existentes: %w", err)
	}
	existingCounts := make(map[string]int, len(existing))
	for _, v := range existing {
		key := fmt.Sprintf("%d:%d:%d:%d", v.IDAvion, v.IDOrigen, v.IDDestino, v.SalidaProgramada)
		existingCounts[key]++
	}

	// Parse location reference for timezone (we use UTC timestamps)
	loc := time.UTC

	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("leer filas CSV: %w", err)
	}

	log.Printf("[Seed Flights] Importing %d flights from CSV...", len(records))

	vuelos := make([]models.Vuelo, 0, len(records))
	skipped := 0

	for i, row := range records {
		if len(row) < 7 {
			skipped++
			continue
		}

		// Parse date: MM/DD/YY
		dateStr := strings.TrimSpace(row[0])
		timeStr := strings.TrimSpace(row[1])
		origin := strings.TrimSpace(row[2])
		dest := strings.TrimSpace(row[3])
		acIDStr := strings.TrimSpace(row[4])
		status := strings.TrimSpace(row[5])
		gateStr := strings.TrimSpace(row[6])

		// Parse aircraft_id
		acID, err := strconv.Atoi(acIDStr)
		if err != nil || !validPlane[uint(acID)] {
			skipped++
			continue
		}

		// Resolve IDs
		originID, okO := ciudadByCode[origin]
		destID, okD := ciudadByCode[dest]
		if !okO || !okD {
			// Unknown airport code — skip
			skipped++
			logRejection(row, fmt.Sprintf("Ciudad de origen (%s) o destino (%s) no existe en la base de datos", origin, dest))
			continue
		}

		estadoID, okS := statusMap[status]
		if !okS {
			estadoID = 1 // default to SCHEDULED
		}

		puertaID, okP := puertaByName[originID][gateStr]
		if !okP {
			puertaID = fallbackGate[originID]
			if puertaID == 0 {
				skipped++
				continue
			}
		}

		avionID := uint(acID)

		// Build the departure Unix timestamp from date + time
		// CSV date is MM/DD/YY, time is H:MM (24h)
		dtStr := fmt.Sprintf("%s %s", dateStr, timeStr)
		t, err := time.ParseInLocation("01/02/06 15:04", dtStr, loc)
		if err != nil {
			// Try with single-digit hour
			t, err = time.ParseInLocation("01/02/06 3:04", dtStr, loc)
			if err != nil {
				skipped++
				continue
			}
		}

		salidaUnix := t.Unix()

		// A fare in either class makes this directed route importable.
		allowed, _, _, hours := matrices.RouteAvailability(origin, dest)
		if !allowed {
			skipped++
			logRejection(row, fmt.Sprintf("La ruta de %s a %s no tiene duración o tarifa en ninguna clase", origin, dest))
			continue
		}
		// Store each CSV flight only on the PostgreSQL node that owns its origin.
		var originCity models.Ciudad
		for _, city := range ciudades {
			if city.ID == originID {
				originCity = city
				break
			}
		}
		if (originCity.Region == "America") != (ownerRegion == "America") {
			continue
		}
		llegadaUnix := salidaUnix + int64(hours*3600)
		key := fmt.Sprintf("%d:%d:%d:%d", acID, originID, destID, salidaUnix)
		if existingCounts[key] > 0 {
			existingCounts[key]--
			continue
		}

		vuelo := models.Vuelo{
			ID:                uint(100000000 + i + 1),
			IDOrigen:          originID,
			IDDestino:         destID,
			IDEstadoVuelo:     estadoID,
			IDPuerta:          puertaID,
			IDAvion:           avionID,
			SalidaProgramada:  salidaUnix,
			LlegadaProgramada: llegadaUnix,
			SalidaReal:        0,
			LlegadaReal:       0,
			FechaSalida:       salidaUnix,
			FechaLlegada:      llegadaUnix,
		}
		if ownerRegion != "America" {
			vuelo.ID += 1000000000
		}
		vuelos = append(vuelos, vuelo)
	}

	// Rule 1: Set future flights to SCHEDULED (estado ID = 1)
	now := time.Now().Unix()
	for i := range vuelos {
		if vuelos[i].SalidaProgramada > now {
			vuelos[i].IDEstadoVuelo = 1
		}
	}

	// Keep valid route data while recording discontinuities separately.
	// Sort flights by aircraft ID, then by scheduled departure time
	sort.Slice(vuelos, func(i, j int) bool {
		if vuelos[i].IDAvion == vuelos[j].IDAvion {
			return vuelos[i].SalidaProgramada < vuelos[j].SalidaProgramada
		}
		return vuelos[i].IDAvion < vuelos[j].IDAvion
	})

	var validVuelos []models.Vuelo

	for _, v := range vuelos {
		// Rule 2 is temporarily disabled as requested by user.
		// We append all flights to validVuelos without checking lastDest.
		validVuelos = append(validVuelos, v)
	}

	vuelos = validVuelos
	log.Printf("[Seed Flights] Continuity does not reject CSV flights; run the diagnostic endpoint for anomalies.")

	if logFile != nil {
		logFile.WriteString(fmt.Sprintf("\n--- RESUMEN FINAL ---\n%d filas nuevas con al menos una clase tarifada. Discontinuidades: diagnóstico separado.\n", len(vuelos)))
	}
	log.Printf("[Seed Flights] Dataset loaded. Starting import of %d valid flights.", len(vuelos))

	// Bulk insert in batches of 500 for performance
	batchSize := 500
	total := 0
	for start := 0; start < len(vuelos); start += batchSize {
		end := start + batchSize
		if end > len(vuelos) {
			end = len(vuelos)
		}
		batch := vuelos[start:end]
		if err := dbConn.Create(&batch).Error; err != nil {
			return total, fmt.Errorf("insertar lote %d: %w", start, err)
		} else {
			total += len(batch)
		}
	}

	log.Printf("[Seed Flights] Successfully imported %d vuelos (%d skipped).", total, skipped)
	// Repair the one-hour placeholder used by earlier versions, in batches
	// by directed route. This does not modify flight IDs or ticket references.
	for from, destinations := range matrices.TravelTime {
		for to, hours := range destinations {
			if hours <= 0 {
				continue
			}
			fromID, fromOK := ciudadByCode[from]
			toID, toOK := ciudadByCode[to]
			if !fromOK || !toOK {
				continue
			}
			delta := int64(hours * 3600)
			dbConn.Model(&models.Vuelo{}).
				Where("id_origen = ? AND id_destino = ? AND llegada_programada <> salida_programada + ?", fromID, toID, delta).
				Updates(map[string]interface{}{
					"llegada_programada": gorm.Expr("salida_programada + ?", delta),
					"fecha_llegada":      gorm.Expr("salida_programada + ?", delta),
				})
		}
	}
	dbConn.Model(&models.Vuelo{}).Where("salida_programada > ?", time.Now().Unix()).Update("id_estado_vuelo", 1)
	return total, nil
}

// CSVPath returns the expected path to the flights CSV dataset.
// Searches in multiple locations to work both locally and inside Docker.
func CSVPath() string {
	candidates := []string{
		// Inside Docker container (mounted volume)
		filepath.Join("/app", "dataset", "02 - Practica 3 Dataset Flights.csv"),
		// Local development - running from backend/
		filepath.Join("..", "dataset", "02 - Practica 3 Dataset Flights.csv"),
		// Local development - running from project root
		filepath.Join("dataset", "02 - Practica 3 Dataset Flights.csv"),
		// Renamed copy in data/ folder
		filepath.Join("data", "flights.csv"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			log.Printf("[Seed Flights] Found CSV at: %s", p)
			return p
		}
	}
	return ""
}
