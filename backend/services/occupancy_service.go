package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"

	"airres-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var seedPassengerNames = []string{
	"María Fernanda Rojas", "José Luis Pabón", "Ana Luísa Silva", "Camila Fernández",
	"Wei Zhang 张伟", "Lǐ Měi 李美", "Haruto Sato 佐藤陽翔", "Yui Tanaka 田中結衣",
	"أحمد محمد", "فاطمة الزهراء", "Omar Al-Khalil", "Layla Haddad",
	"Aarav Patel", "Priya Sharma", "Chinedu Okafor", "Amina Diallo",
	"Sofia Müller", "Léa Dubois", "Mehmet Yılmaz", "Elif Demir",
	"João Pereira", "Inês Costa", "Kwame Mensah", "Olena Kovalenko",
}

var matrixHashMu sync.Mutex
var matrixHash string

func CurrentMatrixHash() string {
	matrixHashMu.Lock()
	defer matrixHashMu.Unlock()
	if matrixHash == "" {
		content, err := os.ReadFile("data/matrices.json")
		if err == nil {
			// The percentages are part of the hash so changing them regenerates manifests.
			soldPct, reservedPct := OccupancyPercentages()
			content = append(content, fmt.Sprintf("|%g|%g", soldPct, reservedPct)...)
			sum := sha256.Sum256(content)
			matrixHash = hex.EncodeToString(sum[:])
		}
	}
	return matrixHash
}

func ResetMatrixHash() {
	matrixHashMu.Lock()
	matrixHash = ""
	matrixHashMu.Unlock()
}

// OccupancyPercentages reads PERCENTAGE_SOLD and PERCENTAGE_RESERVED (defaults
// 73 and 3). Invalid values fall back to the defaults.
func OccupancyPercentages() (sold, reserved float64) {
	sold = percentFromEnv("PERCENTAGE_SOLD", 73)
	reserved = percentFromEnv("PERCENTAGE_RESERVED", 3)
	if sold+reserved > 100 {
		return 73, 3
	}
	return
}

func percentFromEnv(name string, fallback float64) float64 {
	value, err := strconv.ParseFloat(os.Getenv(name), 64)
	if err != nil || value < 0 || value > 100 {
		return fallback
	}
	return value
}

// OccupancyTargets uses the nearest whole seat because capacities such as 228
// cannot represent exactly 73% and 3% with indivisible seats.
func OccupancyTargets(eligible int) (sold, reserved int) {
	if eligible <= 0 {
		return 0, 0
	}
	soldPct, reservedPct := OccupancyPercentages()
	sold = int(math.Round(float64(eligible) * soldPct / 100))
	reserved = int(math.Round(float64(eligible) * reservedPct / 100))
	if sold+reserved > eligible {
		reserved = eligible - sold
	}
	return
}

func BuildFlightManifest(flightID uint, eligible []models.Asiento, active []models.Boleto) models.OcupacionVuelo {
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].ID < eligible[j].ID })
	occupied := make(map[uint]bool, len(active))
	activeSold, activeReserved := 0, 0
	for _, ticket := range active {
		if ticket.Estado == "SALED" {
			activeSold++
			occupied[ticket.IDAsiento] = true
		}
		if ticket.Estado == "RESERVED" || ticket.Estado == "REFUNDED" {
			activeReserved++
			occupied[ticket.IDAsiento] = true
		}
	}
	soldTarget, reservedTarget := OccupancyTargets(len(eligible))
	soldNeeded := soldTarget - activeSold
	reservedNeeded := reservedTarget - activeReserved
	if soldNeeded < 0 {
		soldNeeded = 0
	}
	if reservedNeeded < 0 {
		reservedNeeded = 0
	}
	available := make([]models.Asiento, 0, len(eligible))
	for _, seat := range eligible {
		if !occupied[seat.ID] {
			available = append(available, seat)
		}
	}
	rng := rand.New(rand.NewSource(int64(flightID)))
	rng.Shuffle(len(available), func(i, j int) { available[i], available[j] = available[j], available[i] })
	if soldNeeded > len(available) {
		soldNeeded = len(available)
	}
	if reservedNeeded > len(available)-soldNeeded {
		reservedNeeded = len(available) - soldNeeded
	}
	assignments := make([]models.FlightSeatAssignment, 0, soldNeeded+reservedNeeded)
	for i := 0; i < soldNeeded+reservedNeeded; i++ {
		seat := available[i]
		status := "SALED"
		if i >= soldNeeded {
			status = "RESERVED"
		}
		assignments = append(assignments, models.FlightSeatAssignment{
			SeatID: seat.ID, Status: status,
			PassengerName:  seedPassengerNames[(int(flightID)+i)%len(seedPassengerNames)],
			PassengerEmail: fmt.Sprintf("seed-%d-%d@example.invalid", flightID, seat.ID),
			Passport:       fmt.Sprintf("DEMO-%d-%d", flightID, seat.ID),
		})
	}
	return models.OcupacionVuelo{IDVuelo: flightID, Assignments: assignments, EligibleSeats: len(eligible), SoldCount: soldNeeded, ReservedCount: reservedNeeded}
}

func DecodeManifest(manifest models.OcupacionVuelo) ([]models.FlightSeatAssignment, error) {
	return manifest.Assignments, nil
}

// EnsureFlightOccupancy is safe to call inside a booking transaction. A
// unique flight key prevents two workers from creating separate manifests.
func EnsureFlightOccupancy(tx *gorm.DB, flight models.Vuelo) (models.OcupacionVuelo, error) {
	var current models.OcupacionVuelo
	existingManifest := tx.Where("id_vuelo = ?", flight.ID).Limit(1).Find(&current)
	if existingManifest.Error != nil {
		return current, existingManifest.Error
	}
	if existingManifest.RowsAffected > 0 && current.MatrixHash == CurrentMatrixHash() {
		return current, nil
	}
	var seats []models.Asiento
	if err := tx.Where("id_avion = ?", flight.IDAvion).Find(&seats).Error; err != nil {
		return current, err
	}
	regularFare, regularErr := FareForFlight(tx, &flight, "REGULAR")
	vipFare, vipErr := FareForFlight(tx, &flight, "VIP")
	eligible := make([]models.Asiento, 0, len(seats))
	for _, seat := range seats {
		if (seat.Clase == "REGULAR" && regularErr == nil) || (seat.Clase == "VIP" && vipErr == nil) {
			eligible = append(eligible, seat)
		}
	}
	var existing []models.Boleto
	if err := tx.Where("id_vuelo = ? AND estado IN ?", flight.ID, []string{"SALED", "RESERVED", "REFUNDED"}).Find(&existing).Error; err != nil {
		return current, err
	}
	manifest := BuildFlightManifest(flight.ID, eligible, existing)
	manifest.MatrixHash = CurrentMatrixHash()
	seatByID := make(map[uint]models.Asiento, len(eligible))
	for _, seat := range eligible {
		seatByID[seat.ID] = seat
	}
	entries, _ := DecodeManifest(manifest)
	for _, entry := range entries {
		if entry.Status != "SALED" {
			continue
		}
		if seatByID[entry.SeatID].Clase == "VIP" {
			manifest.FirstIncome += vipFare
		} else {
			manifest.EconomyIncome += regularFare
		}
	}
	manifest.LamportClock = GlobalLamportClock.Tick()
	manifest.VectorClock = TickVectorClock()
	manifest.SourceNode = NodeID
	if existingManifest.RowsAffected > 0 {
		if err := tx.Save(&manifest).Error; err != nil {
			return current, err
		}
		if err := QueueOutboxEvent(tx, "UPDATE", "OcupacionVuelo", &manifest); err != nil {
			return current, err
		}
		return manifest, nil
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&manifest)
	if result.Error != nil {
		return current, result.Error
	}
	if result.RowsAffected > 0 {
		if err := QueueOutboxEvent(tx, "CREATE", "OcupacionVuelo", &manifest); err != nil {
			return current, err
		}
		return manifest, nil
	}
	if err := tx.First(&current, "id_vuelo = ?", flight.ID).Error; err != nil {
		return current, err
	}
	return current, nil
}
