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
	EconomyFares    map[string]map[string]*float64 `json:"economy_fares"`
	FirstClassFares map[string]map[string]*float64 `json:"first_class_fares"`
}



// SeedFlightsFromCSV loads the dataset CSV into the DB if the vuelos table is empty
func SeedFlightsFromCSV(dbConn *gorm.DB, csvPath string) {
	if dbConn == nil {
		return
	}

	// Check if already seeded
	var count int64
	dbConn.Model(&models.Vuelo{}).Count(&count)
	if count > 0 {
		log.Printf("[Seed Flights] Vuelos already seeded (%d rows), skipping CSV import.", count)
		return
	}

	// Open CSV
	f, err := os.Open(csvPath)
	if err != nil {
		log.Printf("[Seed Flights] WARNING: Could not open CSV at %s: %v", csvPath, err)
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true

	// Load matrices.json to validate routes (Rule 3)
	var matrices MatricesJSON
	matrixBytes, err := os.ReadFile(filepath.Join("data", "matrices.json"))
	if err == nil {
		json.Unmarshal(matrixBytes, &matrices)
	} else {
		log.Printf("[Seed Flights] Warning: could not read matrices.json: %v", err)
	}

	// Read header
	if _, err := reader.Read(); err != nil {
		log.Printf("[Seed Flights] Error reading CSV header: %v", err)
		return
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

	// Build puerta lookup: puerta_str -> id
	var puertas []models.Puerta
	dbConn.Find(&puertas)
	puertaByName := make(map[string]uint)
	for _, p := range puertas {
		puertaByName[p.Puerta] = p.ID
	}

	// Parse location reference for timezone (we use UTC timestamps)
	loc := time.UTC

	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[Seed Flights] Error reading CSV records: %v", err)
		return
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
		origin  := strings.TrimSpace(row[2])
		dest    := strings.TrimSpace(row[3])
		acIDStr := strings.TrimSpace(row[4])
		status  := strings.TrimSpace(row[5])
		gateStr := strings.TrimSpace(row[6])

		// Parse aircraft_id
		acID, err := strconv.Atoi(acIDStr)
		if err != nil {
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

		puertaID, okP := puertaByName[gateStr]
		if !okP {
			// Try matching by gate number (e.g. "G26" -> look for any puerta)
			// We'll use a fallback: puerta index mod available puertas
			puertaID = uint((i % len(puertas)) + 1)
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

		// Estimate arrival: salida + travel_time (we don't store it here, it's in the matrix)
		// We'll just set llegada_programada = salida + 1h as placeholder (actual time is in matrix)
		llegadaUnix := salidaUnix + 3600

		// Rule 3: Check if route is supported in matrices.json
		if matrices.EconomyFares != nil {
			ecoFare := matrices.EconomyFares[origin][dest]
			firstFare := matrices.FirstClassFares[origin][dest]
			if ecoFare == nil || firstFare == nil {
				// Route not supported by matrix, drop flight
				skipped++
				logRejection(row, fmt.Sprintf("La ruta de %s a %s no tiene precio en la matriz (null)", origin, dest))
				continue
			}
		}

		vuelo := models.Vuelo{
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
		vuelos = append(vuelos, vuelo)
	}

	// Rule 1: Set future flights to SCHEDULED (estado ID = 1)
	now := time.Now().Unix()
	for i := range vuelos {
		if vuelos[i].SalidaProgramada > now {
			vuelos[i].IDEstadoVuelo = 1
		}
	}

	// Rule 2: Check continuity (no magic plane jumps)
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
	log.Printf("[Seed Flights] Validated flights. Magic jump check (Rule 2) is currently DISABLED.")

	if logFile != nil {
		logFile.WriteString(fmt.Sprintf("\n--- RESUMEN FINAL ---\n%d vuelos sobrevivieron a la revisión de ciudades y matriz, y fueron importados exitosamente (Saltos mágicos permitidos).\n", len(vuelos)))
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
			log.Printf("[Seed Flights] Error inserting batch starting at %d: %v", start, err)
		} else {
			total += len(batch)
		}
	}

	log.Printf("[Seed Flights] Successfully imported %d vuelos (%d skipped).", total, skipped)
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
