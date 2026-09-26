package data

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GenerateRejectedCSV writes only rows rejected by the airport list or
// directed fare/time matrix. It preserves every source column for auditing.
func GenerateRejectedCSV(sourcePath, matrixPath, outputPath string) (int, error) {
	matrixBytes, err := os.ReadFile(matrixPath)
	if err != nil {
		return 0, err
	}
	var matrix MatricesJSON
	if err := json.Unmarshal(matrixBytes, &matrix); err != nil {
		return 0, err
	}
	validCity := make(map[string]bool, len(matrix.Airports))
	for _, airport := range matrix.Airports {
		validCity[airport] = true
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	reader := csv.NewReader(source)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, err
	}
	output, err := os.Create(outputPath)
	if err != nil {
		return 0, err
	}
	defer output.Close()
	writer := csv.NewWriter(output)
	if err := writer.Write(append(append([]string{}, header...), "motivo_rechazo")); err != nil {
		return 0, err
	}
	count := 0
	for {
		row, readErr := reader.Read()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return count, readErr
		}
		if len(row) < 4 {
			continue
		}
		origin, dest := strings.TrimSpace(row[2]), strings.TrimSpace(row[3])
		reason := ""
		switch {
		case !validCity[origin] && !validCity[dest]:
			reason = fmt.Sprintf("Origen %s y destino %s fuera de la lista de aeropuertos", origin, dest)
		case !validCity[origin]:
			reason = fmt.Sprintf("Origen %s fuera de la lista de aeropuertos", origin)
		case !validCity[dest]:
			reason = fmt.Sprintf("Destino %s fuera de la lista de aeropuertos", dest)
		case origin == dest:
			reason = "Origen y destino son el mismo aeropuerto"
		default:
			economy, first := matrix.EconomyFares[origin][dest], matrix.FirstClassFares[origin][dest]
			if (economy == nil || *economy <= 0) && (first == nil || *first <= 0) {
				reason = "Sin precio en Primera Clase ni Clase Turista para esta dirección"
			}
			if (matrix.TravelTime[origin][dest] == nil || *matrix.TravelTime[origin][dest] <= 0) && reason == "" {
				reason = "Sin duración positiva en la matriz para esta dirección"
			}
		}
		if reason != "" {
			if err := writer.Write(append(append([]string{}, row...), reason)); err != nil {
				return count, err
			}
			count++
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return count, err
	}
	return count, nil
}
