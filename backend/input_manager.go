package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"airres-api/data"
	"airres-api/db"
	"airres-api/models"
	"airres-api/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

type inputJob struct {
	ID        string   `json:"id"`
	State     string   `json:"state"`
	Step      string   `json:"step"`
	Percent   int      `json:"percent"`
	Error     string   `json:"error,omitempty"`
	Rows      int      `json:"rows"`
	Eligible  int      `json:"eligible"`
	Rejected  int      `json:"rejected"`
	America   int      `json:"america"`
	Other     int      `json:"other"`
	Files     []string `json:"files"`
	StagePath string   `json:"-"`
}

var inputJobs = struct {
	sync.Mutex
	jobs    map[string]*inputJob
	running bool
}{jobs: map[string]*inputJob{}}

func jobSnapshot(id string) (inputJob, bool) {
	inputJobs.Lock()
	defer inputJobs.Unlock()
	job, ok := inputJobs.jobs[id]
	if !ok {
		return inputJob{}, false
	}
	copy := *job
	copy.Files = append([]string(nil), job.Files...)
	return copy, true
}

func updateJob(id string, state string, percent int, step string, err error) {
	inputJobs.Lock()
	defer inputJobs.Unlock()
	job := inputJobs.jobs[id]
	job.State, job.Step = state, step
	if percent >= 0 {
		job.Percent = percent
	}
	if err != nil {
		job.Error = err.Error()
	}
	if state == "completed" || state == "failed" {
		inputJobs.running = false
	}
}

func uploaded(c *gin.Context, field string, limit int64) ([]byte, string, bool, error) {
	fh, err := c.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return nil, "", false, nil
	}
	if err != nil {
		return nil, "", false, err
	}
	file, err := fh.Open()
	if err != nil {
		return nil, "", false, err
	}
	defer file.Close()
	content, err := data.ReadLimited(file, limit)
	return content, fh.Filename, true, err
}

func previewInputs(c *gin.Context) {
	inputJobs.Lock()
	running := inputJobs.running
	inputJobs.Unlock()
	if running {
		c.JSON(409, gin.H{"error": "Espera a que termine el procesamiento actual"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 80<<20)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pudieron leer los archivos: " + err.Error()})
		return
	}
	activeMatrix, err := os.ReadFile(filepath.Join("data", "matrices.json"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var matrix data.MatricesJSON
	if err := json.Unmarshal(activeMatrix, &matrix); err != nil {
		c.JSON(500, gin.H{"error": "Matriz activa inválida"})
		return
	}
	files := []string{}
	if content, name, present, err := uploaded(c, "matrices", 10<<20); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	} else if present {
		if strings.ToLower(filepath.Ext(name)) != ".json" {
			c.JSON(400, gin.H{"error": "Las tres matrices juntas deben estar en JSON"})
			return
		}
		matrix, err = data.MatrixFromJSON(content)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		files = append(files, "Tres matrices: "+name)
	}
	for _, field := range []string{"travel_time", "economy_fares", "first_class_fares"} {
		content, name, present, err := uploaded(c, field, 10<<20)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if !present {
			continue
		}
		grid, err := data.MatrixGrid(content, name, field)
		if err != nil {
			c.JSON(400, gin.H{"error": field + ": " + err.Error()})
			return
		}
		switch field {
		case "travel_time":
			matrix.TravelTime = data.TimeGrid(grid)
		case "economy_fares":
			matrix.EconomyFares = grid
		case "first_class_fares":
			matrix.FirstClassFares = grid
		}
		files = append(files, field+": "+name)
	}
	if err := data.ValidateMatrices(matrix); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	metadata, err := os.ReadFile(filepath.Join("data", "airports.json"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var airports map[string]airportMetadata
	if err := json.Unmarshal(metadata, &airports); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	for _, code := range matrix.Airports {
		meta, ok := airports[code]
		if !ok || meta.Country == "" || meta.Region == "" || meta.TimeZone == "" {
			c.JSON(400, gin.H{"error": "Falta país, región o zona horaria para " + code + " en airports.json"})
			return
		}
		if _, err := time.LoadLocation(meta.TimeZone); err != nil {
			c.JSON(400, gin.H{"error": "Zona horaria inválida para " + code})
			return
		}
	}
	var dataset []byte
	if content, name, present, err := uploaded(c, "dataset", 50<<20); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	} else if present {
		dataset, _, err = data.DatasetCSV(content, name)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		files = append(files, "Dataset: "+name)
	} else {
		path := data.CSVPath()
		if path == "" {
			c.JSON(400, gin.H{"error": "No hay dataset activo; sube un CSV o Excel"})
			return
		}
		original, err := os.ReadFile(path)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		dataset, _, err = data.DatasetCSV(original, path)
		if err != nil {
			c.JSON(400, gin.H{"error": "Dataset activo inválido: " + err.Error()})
			return
		}
	}
	if len(files) == 0 {
		c.JSON(400, gin.H{"error": "Selecciona al menos un archivo nuevo"})
		return
	}
	var planeIDs []uint
	if err := db.PGAmerica.Model(&models.Avion{}).Pluck("id", &planeIDs).Error; err != nil {
		c.JSON(503, gin.H{"error": "No se pudo consultar el catálogo de aviones"})
		return
	}
	planes := make(map[uint]bool, len(planeIDs))
	for _, id := range planeIDs {
		planes[id] = true
	}
	rows, eligible, rejected, err := previewDataset(dataset, matrix, planes)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if eligible == 0 {
		c.JSON(400, gin.H{"error": "Ningún vuelo es utilizable con las matrices y aeropuertos actuales"})
		return
	}
	stage, err := os.MkdirTemp("uploads", "inputs-")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	encoded, _ := json.MarshalIndent(matrix, "", "  ")
	if err := os.WriteFile(filepath.Join(stage, "matrices.json"), encoded, 0600); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(filepath.Join(stage, "flights.csv"), dataset, 0600); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	job := &inputJob{ID: uuid.NewString(), State: "preview", Step: "Archivos validados", Percent: 0, Rows: rows, Eligible: eligible, Rejected: rejected, Files: files, StagePath: stage}
	inputJobs.Lock()
	inputJobs.jobs[job.ID] = job
	inputJobs.Unlock()
	c.JSON(200, job)
}

func previewDataset(content []byte, matrix data.MatricesJSON, planes map[uint]bool) (int, int, int, error) {
	r := csv.NewReader(bytes.NewReader(content))
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil {
		return 0, 0, 0, err
	}
	rows, eligible, rejected := 0, 0, 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, 0, err
		}
		rows++
		if len(row) < 7 {
			rejected++
			continue
		}
		origin, dest := strings.ToUpper(strings.TrimSpace(row[2])), strings.ToUpper(strings.TrimSpace(row[3]))
		allowed, _, _, _ := matrix.RouteAvailability(origin, dest)
		plane, planeErr := strconv.Atoi(strings.TrimSpace(row[4]))
		dateTime := strings.TrimSpace(row[0]) + " " + strings.TrimSpace(row[1])
		_, dateErr := time.ParseInLocation("01/02/06 15:04", dateTime, time.UTC)
		if dateErr != nil {
			_, dateErr = time.ParseInLocation("01/02/06 3:04", dateTime, time.UTC)
		}
		if allowed && planeErr == nil && plane > 0 && planes[uint(plane)] && dateErr == nil {
			eligible++
		} else {
			rejected++
		}
	}
	return rows, eligible, rejected, nil
}

func getInputJob(c *gin.Context) {
	job, ok := jobSnapshot(c.Param("id"))
	if !ok {
		c.JSON(404, gin.H{"error": "Proceso no encontrado"})
		return
	}
	c.JSON(200, job)
}

func startInputJob(c *gin.Context) {
	id := c.Param("id")
	inputJobs.Lock()
	job, ok := inputJobs.jobs[id]
	if !ok {
		inputJobs.Unlock()
		c.JSON(404, gin.H{"error": "Proceso no encontrado"})
		return
	}
	if inputJobs.running || job.State != "preview" {
		inputJobs.Unlock()
		c.JSON(409, gin.H{"error": "Ya hay un proceso activo o este ya fue iniciado"})
		return
	}
	inputJobs.running = true
	job.State, job.Step, job.Percent = "running", "Preparando reemplazo", 5
	inputJobs.Unlock()
	go func() {
		if err := runInputJob(id); err != nil {
			updateJob(id, "failed", -1, "Error durante el reemplazo", err)
		}
	}()
	c.JSON(202, gin.H{"id": id, "state": "running"})
}

func resetPostgres(conn *gorm.DB) error {
	for _, query := range []string{
		"DELETE FROM sync_outbox", "DELETE FROM ocupaciones_vuelo", "DELETE FROM boletos", "DELETE FROM vuelos",
		"DELETE FROM id_allocators WHERE name LIKE 'ticket_%' OR name LIKE 'flight_%'",
	} {
		if err := conn.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func replaceFile(source, dest string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, content, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

func verifyRelationalMatrices(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var expected data.MatricesJSON
	if err := json.Unmarshal(content, &expected); err != nil {
		return err
	}
	for name, conn := range map[string]*gorm.DB{"América": db.PGAmerica, "Europa/Asia": db.PGEuropaAsia} {
		var prices models.Precios
		var details models.DetallesVuelos
		if err := conn.First(&prices).Error; err != nil {
			return fmt.Errorf("precios de %s: %w", name, err)
		}
		if err := conn.First(&details).Error; err != nil {
			return fmt.Errorf("tiempos de %s: %w", name, err)
		}
		var economy, first map[string]map[string]*float64
		var times map[string]map[string]float64
		if err := json.Unmarshal(prices.MatrizPreciosRegular, &economy); err != nil {
			return err
		}
		if err := json.Unmarshal(prices.MatrizPreciosVip, &first); err != nil {
			return err
		}
		if err := json.Unmarshal(details.MatrizTiempos, &times); err != nil {
			return err
		}
		if !reflect.DeepEqual(economy, expected.EconomyFares) || !reflect.DeepEqual(first, expected.FirstClassFares) || !reflect.DeepEqual(times, expected.TravelTime) {
			return fmt.Errorf("las matrices de %s no coinciden con los archivos activos", name)
		}
	}
	return nil
}

func runInputJob(id string) error {
	services.InputImportActive.Store(true)
	defer services.InputImportActive.Store(false)
	job, ok := jobSnapshot(id)
	if !ok {
		return errors.New("proceso no encontrado")
	}
	if !db.IsAvailable(db.PGAmerica) || !db.IsAvailable(db.PGEuropaAsia) || !db.IsMongoAvailable() {
		return errors.New("se necesitan los dos nodos PostgreSQL y MongoDB activos")
	}
	path := data.CSVPath()
	if path == "" {
		return errors.New("no se encontró el dataset activo")
	}
	matrixPath := filepath.Join("data", "matrices.json")
	oldMatrix, err := os.ReadFile(matrixPath)
	if err != nil {
		return err
	}
	oldDataset, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	replacedFiles, committing := false, false
	defer func() {
		if replacedFiles && !committing {
			_ = os.WriteFile(matrixPath, oldMatrix, 0644)
			_ = os.WriteFile(path, oldDataset, 0644)
			services.ResetMatrixHash()
		}
	}()
	updateJob(id, "running", 12, "Activando los archivos nuevos", nil)
	if err := replaceFile(filepath.Join(job.StagePath, "matrices.json"), matrixPath); err != nil {
		return err
	}
	replacedFiles = true
	services.ResetMatrixHash()
	if err := replaceFile(filepath.Join(job.StagePath, "flights.csv"), path); err != nil {
		return err
	}
	seedAirportCatalog()
	updateJob(id, "running", 25, "Preparando reemplazo en PostgreSQL", nil)
	txAM := db.PGAmerica.Begin()
	if txAM.Error != nil {
		return txAM.Error
	}
	defer txAM.Rollback()
	txEU := db.PGEuropaAsia.Begin()
	if txEU.Error != nil {
		return txEU.Error
	}
	defer txEU.Rollback()
	if err := resetPostgres(txAM); err != nil {
		return fmt.Errorf("PostgreSQL América: %w", err)
	}
	if err := resetPostgres(txEU); err != nil {
		return fmt.Errorf("PostgreSQL Europa/Asia: %w", err)
	}
	updateJob(id, "running", 40, "Procesando vuelos de América", nil)
	am, err := data.ImportFlightsFromCSV(txAM, path, matrixPath, "America")
	if err != nil {
		return err
	}
	updateJob(id, "running", 60, "Procesando vuelos de Europa y Asia", nil)
	eu, err := data.ImportFlightsFromCSV(txEU, path, matrixPath, "EuropaAsia")
	if err != nil {
		return err
	}
	if _, err := data.GenerateRejectedCSV(path, matrixPath, filepath.Join("reports", "vuelos_rechazados.csv")); err != nil {
		return fmt.Errorf("generar reporte de rechazos: %w", err)
	}
	updateJob(id, "running", 75, "Confirmando reemplazo y actualizando precios", nil)
	committing = true
	if err := txAM.Commit().Error; err != nil {
		return err
	}
	if err := txEU.Commit().Error; err != nil {
		return err
	}
	seedMatrices(db.PGAmerica)
	seedMatrices(db.PGEuropaAsia)
	seedPrecios(db.PGAmerica)
	seedPrecios(db.PGEuropaAsia)
	if err := verifyRelationalMatrices(matrixPath); err != nil {
		return err
	}
	seedAsientos(db.PGAmerica)
	seedAsientos(db.PGEuropaAsia)
	seedMongoMatrices()
	inputJobs.Lock()
	inputJobs.jobs[id].America, inputJobs.jobs[id].Other = am, eu
	inputJobs.Unlock()
	updateJob(id, "running", 80, "Limpiando y sincronizando MongoDB", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	for _, collection := range []string{"vuelos", "boletos", "ocupaciones_vuelo"} {
		if _, err := db.MongoDatabase.Collection(collection).DeleteMany(ctx, bson.M{}); err != nil {
			return err
		}
	}
	seedDemoFlights()
	bootstrapMongo(true)
	var amCount, euCount int64
	if err := db.PGAmerica.Model(&models.Vuelo{}).Where("id < ?", 1000000000).Count(&amCount).Error; err != nil {
		return err
	}
	if err := db.PGEuropaAsia.Model(&models.Vuelo{}).Where("id >= ?", 1000000000).Count(&euCount).Error; err != nil {
		return err
	}
	mongoCount, err := db.MongoDatabase.Collection("vuelos").CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if mongoCount != amCount+euCount {
		return fmt.Errorf("MongoDB tiene %d vuelos; se esperaban %d", mongoCount, amCount+euCount)
	}
	info, _ := json.Marshal(gin.H{"files": job.Files, "updated_at": time.Now().UTC().Format(time.RFC3339), "rows": job.Rows})
	if err := os.WriteFile(filepath.Join("data", "input-info.json"), info, 0644); err != nil {
		return err
	}
	services.InputImportActive.Store(false)
	updateJob(id, "completed", 100, "Procesamiento completo. Los datos nuevos ya están disponibles.", nil)
	go seedMissingOccupancy()
	return nil
}

func currentInputs(c *gin.Context) {
	path := data.CSVPath()
	matrixBytes, err := os.ReadFile(filepath.Join("data", "matrices.json"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var matrix data.MatricesJSON
	if err := json.Unmarshal(matrixBytes, &matrix); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var rows int
	if path != "" {
		if f, err := os.Open(path); err == nil {
			r := csv.NewReader(f)
			r.FieldsPerRecord = -1
			for {
				if _, err := r.Read(); err != nil {
					break
				}
				rows++
			}
			f.Close()
		}
	}
	if rows > 0 {
		rows--
	}
	result := gin.H{"dataset": filepath.Base(path), "rows": rows, "airports": len(matrix.Airports), "matrix": "matrices.json"}
	if info, err := os.ReadFile(filepath.Join("data", "input-info.json")); err == nil {
		var saved map[string]interface{}
		if json.Unmarshal(info, &saved) == nil {
			result["last_import"] = saved
		}
	}
	c.JSON(200, result)
}

func registerInputRoutes(r *gin.Engine) {
	r.GET("/api/entradas", currentInputs)
	r.GET("/api/entradas/rechazos.csv", func(c *gin.Context) {
		path := filepath.Join("reports", "vuelos_rechazados.csv")
		if _, err := os.Stat(path); err != nil {
			c.JSON(404, gin.H{"error": "Reporte no disponible"})
			return
		}
		c.FileAttachment(path, "vuelos_rechazados.csv")
	})
	r.POST("/api/entradas/preview", previewInputs)
	r.GET("/api/entradas/jobs/:id", getInputJob)
	r.POST("/api/entradas/jobs/:id/start", startInputJob)
}
