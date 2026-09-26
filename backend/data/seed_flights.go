package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"airres-api/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MatricesJSON maps to matrices.json to check supported routes
type MatricesJSON struct {
	Airports        []string                       `json:"airports"`
	TravelTime      map[string]map[string]*float64 `json:"travel_time"`
	EconomyFares    map[string]map[string]*float64 `json:"economy_fares"`
	FirstClassFares map[string]map[string]*float64 `json:"first_class_fares"`
}

// RouteAvailability checks the selected direction. A route may sell either
// class independently, but it must have a positive flight duration.
func (m MatricesJSON) RouteAvailability(origin, destination string) (allowed, economy, vip bool, hours float64) {
	if origin == destination {
		return false, false, false, 0
	}
	if duration := m.TravelTime[origin][destination]; duration != nil {
		hours = *duration
	}
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
	codeByID := make(map[uint]string)
	americanCity := make(map[uint]bool)
	for _, c := range ciudades {
		ciudadByCode[c.Codigo] = c.ID
		codeByID[c.ID] = c.Codigo
		americanCity[c.ID] = c.Region == "America"
	}
	var planes []models.Avion
	dbConn.Select("id").Find(&planes)
	validPlane := make(map[uint]bool, len(planes))
	planeIDs := make([]uint, 0, len(planes))
	for _, plane := range planes {
		validPlane[plane.ID] = true
		planeIDs = append(planeIDs, plane.ID)
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

	// Flight IDs are deterministic, so rows imported earlier are skipped by ID.
	var existingIDs []uint
	if err := dbConn.Model(&models.Vuelo{}).Pluck("id", &existingIDs).Error; err != nil {
		return 0, fmt.Errorf("consultar vuelos existentes: %w", err)
	}
	existing := make(map[uint]bool, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = true
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
		llegadaUnix := salidaUnix + int64(hours*3600)
		estadoID := FlightStatusAt(salidaUnix, llegadaUnix, time.Now().Unix())

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
		// The ID depends on the origin's region so both regional imports assign
		// aircraft over the same IDs and reach identical results.
		if !americanCity[originID] {
			vuelo.ID += 1000000000
		}
		vuelos = append(vuelos, vuelo)
	}

	// Assign aircraft across every CSV row, including flights owned by the other
	// region, so aircraft never overlap or appear at an airport they did not fly to.
	hoursBetween := func(from, to uint) float64 {
		if duration := matrices.TravelTime[codeByID[from]][codeByID[to]]; duration != nil {
			return *duration
		}
		return 0
	}
	vuelos, ferries, conflicts := AssignAircraft(vuelos, planeIDs, hoursBetween)
	for _, rejected := range conflicts {
		logRejection([]string{fmt.Sprint(rejected.ID)}, "Ningún avión puede estar en el aeropuerto de origen a tiempo")
	}
	owned := vuelos[:0]
	for _, flight := range vuelos {
		if americanCity[flight.IDOrigen] == (ownerRegion == "America") && !existing[flight.ID] {
			owned = append(owned, flight)
		}
	}
	vuelos = owned

	// Every region stores the full positioning schedule; it is identical in both.
	for start := 0; start < len(ferries); start += 500 {
		batch := ferries[start:min(start+500, len(ferries))]
		if err := dbConn.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch).Error; err != nil {
			return 0, fmt.Errorf("insertar reposicionamientos: %w", err)
		}
	}

	if logFile != nil {
		logFile.WriteString(fmt.Sprintf("\n--- RESUMEN FINAL ---\n%d vuelos admitidos; %d rechazados por falta de avión; %d reposicionamientos.\n", len(vuelos), len(conflicts), len(ferries)))
	}
	log.Printf("[Seed Flights] Dataset loaded. Starting import of %d valid flights (%d repositionings).", len(vuelos), len(ferries))

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
		for to, duration := range destinations {
			if duration == nil || *duration <= 0 {
				continue
			}
			fromID, fromOK := ciudadByCode[from]
			toID, toOK := ciudadByCode[to]
			if !fromOK || !toOK {
				continue
			}
			delta := int64(*duration * 3600)
			dbConn.Model(&models.Vuelo{}).
				Where("id_origen = ? AND id_destino = ? AND llegada_programada <> salida_programada + ?", fromID, toID, delta).
				Updates(map[string]interface{}{
					"llegada_programada": gorm.Expr("salida_programada + ?", delta),
					"fecha_llegada":      gorm.Expr("salida_programada + ?", delta),
				})
		}
	}
	// Refresh imported flights from timestamps, including rows imported previously.
	now := time.Now().Unix()
	firstImportedID := uint(100000001)
	if ownerRegion != "America" {
		firstImportedID += 1000000000
	}
	if err := dbConn.Model(&models.Vuelo{}).Where("id >= ? AND id < ? AND llegada_programada <= ?", firstImportedID, firstImportedID+200000, now).Update("id_estado_vuelo", 5).Error; err != nil {
		return total, err
	}
	if err := dbConn.Model(&models.Vuelo{}).Where("id >= ? AND id < ? AND salida_programada <= ? AND llegada_programada > ?", firstImportedID, firstImportedID+200000, now, now).Update("id_estado_vuelo", 4).Error; err != nil {
		return total, err
	}
	if err := dbConn.Model(&models.Vuelo{}).Where("id >= ? AND id < ? AND salida_programada > ?", firstImportedID, firstImportedID+200000, now).Update("id_estado_vuelo", 1).Error; err != nil {
		return total, err
	}
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
