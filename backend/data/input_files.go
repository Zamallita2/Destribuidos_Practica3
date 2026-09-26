package data

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

var FlightColumns = []string{"flight_date", "flight_time", "origin", "destination", "aircraft_id", "status", "gate"}

var columnAliases = map[string]string{
	"fecha": "flight_date", "hora": "flight_time", "origen": "origin", "destino": "destination",
	"id_avion": "aircraft_id", "avion": "aircraft_id", "estado": "status", "puerta": "gate",
}

// SpreadsheetRows reads the first worksheet. Matrix sheets use airport codes
// in row one and column one; dataset sheets use the seven CSV column names.
func SpreadsheetRows(content []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: 256 << 20, UnzipXMLSizeLimit: 16 << 20})
	if err != nil {
		return nil, fmt.Errorf("Excel inválido: %w", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("el Excel no tiene hojas")
	}
	return f.GetRows(sheets[0])
}

func rowsForFile(content []byte, filename string) ([][]string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".xlsx":
		return SpreadsheetRows(content)
	case ".csv":
		r := csv.NewReader(bytes.NewReader(content))
		r.TrimLeadingSpace = true
		r.FieldsPerRecord = -1
		return r.ReadAll()
	default:
		return nil, errors.New("usa un archivo CSV o Excel .xlsx")
	}
}

func DatasetCSV(content []byte, filename string) ([]byte, int, error) {
	rows, err := rowsForFile(content, filename)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) < 2 {
		return nil, 0, errors.New("el dataset debe tener encabezado y al menos una fila")
	}
	indices := map[string]int{}
	for index, heading := range rows[0] {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(heading, "\ufeff")))
		if alias, ok := columnAliases[name]; ok {
			name = alias
		}
		if _, exists := indices[name]; exists {
			return nil, 0, fmt.Errorf("columna repetida: %s", name)
		}
		indices[name] = index
	}
	for _, name := range FlightColumns {
		if _, ok := indices[name]; !ok {
			return nil, 0, fmt.Errorf("falta la columna %s", name)
		}
	}
	if len(rows) > 200001 {
		return nil, 0, errors.New("el dataset supera 200000 filas")
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(FlightColumns); err != nil {
		return nil, 0, err
	}
	for n, row := range rows[1:] {
		ordered := make([]string, len(FlightColumns))
		for i, name := range FlightColumns {
			index := indices[name]
			if index >= len(row) {
				if name != "gate" {
					return nil, 0, fmt.Errorf("fila %d: falta %s", n+2, name)
				}
				continue
			}
			ordered[i] = strings.TrimSpace(row[index])
		}
		ordered[2] = strings.ToUpper(ordered[2])
		ordered[3] = strings.ToUpper(ordered[3])
		ordered[5] = strings.ToUpper(ordered[5])
		if err := w.Write(ordered); err != nil {
			return nil, 0, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(rows) - 1, nil
}

func MatrixFromJSON(content []byte) (MatricesJSON, error) {
	var raw struct {
		Airports        []string        `json:"airports"`
		TravelTime      json.RawMessage `json:"travel_time"`
		EconomyFares    json.RawMessage `json:"economy_fares"`
		FirstClassFares json.RawMessage `json:"first_class_fares"`
	}
	var matrix MatricesJSON
	if err := json.Unmarshal(content, &raw); err != nil {
		return matrix, fmt.Errorf("JSON de matrices inválido: %w", err)
	}
	matrix.Airports = raw.Airports
	var err error
	if matrix.TravelTime, err = parseJSONGrid(raw.TravelTime, "travel_time"); err != nil {
		return matrix, err
	}
	if matrix.EconomyFares, err = parseJSONGrid(raw.EconomyFares, "economy_fares"); err != nil {
		return matrix, err
	}
	if matrix.FirstClassFares, err = parseJSONGrid(raw.FirstClassFares, "first_class_fares"); err != nil {
		return matrix, err
	}
	return matrix, ValidateMatrices(matrix)
}

func parseJSONGrid(content []byte, kind string) (map[string]map[string]*float64, error) {
	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("JSON de matriz inválido: %w", err)
	}
	grid := make(map[string]map[string]*float64, len(raw))
	for origin, row := range raw {
		grid[origin] = make(map[string]*float64, len(row))
		for dest, cell := range row {
			var value any
			decoder := json.NewDecoder(bytes.NewReader(cell))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return nil, err
			}
			if parsed, ok := matrixValue(fmt.Sprint(value), kind); ok {
				grid[origin][dest] = &parsed
			} else {
				grid[origin][dest] = nil
			}
		}
	}
	return grid, nil
}

func matrixValue(raw, kind string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if kind != "travel_time" {
		mantissa := strings.SplitN(strings.ToLower(raw), "e", 2)[0]
		if parts := strings.SplitN(mantissa, ".", 2); len(parts) == 2 && len(parts[1]) > 2 {
			return 0, false
		}
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, false
	}
	if kind != "travel_time" && math.Abs(value*100-math.Round(value*100)) > 1e-7 {
		return 0, false
	}
	return value, true
}

func MatrixGrid(content []byte, filename string, kind string) (map[string]map[string]*float64, error) {
	if strings.EqualFold(filepath.Ext(filename), ".json") {
		return parseJSONGrid(content, kind)
	}
	rows, err := rowsForFile(content, filename)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 || len(rows[0]) < 2 {
		return nil, errors.New("la matriz necesita códigos de aeropuerto en la primera fila y columna")
	}
	columns := rows[0][1:]
	result := make(map[string]map[string]*float64, len(rows)-1)
	for rowIndex, row := range rows[1:] {
		if len(row) > len(columns)+1 || len(row) == 0 {
			return nil, fmt.Errorf("fila %d: número de columnas distinto del encabezado", rowIndex+2)
		}
		for len(row) < len(columns)+1 {
			row = append(row, "")
		}
		origin := strings.ToUpper(strings.TrimSpace(row[0]))
		if origin == "" || result[origin] != nil {
			return nil, fmt.Errorf("fila %d: origen vacío o repetido", rowIndex+2)
		}
		result[origin] = make(map[string]*float64, len(columns))
		for colIndex, raw := range row[1:] {
			dest := strings.ToUpper(strings.TrimSpace(columns[colIndex]))
			if dest == "" {
				return nil, fmt.Errorf("columna %d: destino vacío", colIndex+2)
			}
			raw = strings.TrimSpace(raw)
			if raw == "" || raw == "-" || strings.EqualFold(raw, "null") {
				result[origin][dest] = nil
				continue
			}
			if value, ok := matrixValue(raw, kind); ok {
				result[origin][dest] = &value
			} else {
				result[origin][dest] = nil
			}
		}
	}
	return result, nil
}

func ValidateMatrices(matrix MatricesJSON) error {
	if len(matrix.Airports) < 2 {
		return errors.New("se necesitan al menos dos aeropuertos")
	}
	seen := map[string]bool{}
	for _, code := range matrix.Airports {
		if len(code) != 3 || strings.ToUpper(code) != code || seen[code] {
			return fmt.Errorf("código de aeropuerto inválido o repetido: %s", code)
		}
		seen[code] = true
	}
	for _, part := range []struct {
		name string
		grid map[string]map[string]*float64
	}{
		{"tiempos", matrix.TravelTime},
		{"turista", matrix.EconomyFares},
		{"primera clase", matrix.FirstClassFares},
	} {
		for _, origin := range matrix.Airports {
			row := part.grid[origin]
			if row == nil {
				return fmt.Errorf("matriz de %s: falta fila %s", part.name, origin)
			}
			for _, dest := range matrix.Airports {
				value, ok := row[dest]
				if !ok {
					return fmt.Errorf("matriz de %s: falta %s→%s", part.name, origin, dest)
				}
				if value != nil && (*value < 0 || math.IsNaN(*value) || math.IsInf(*value, 0)) {
					return fmt.Errorf("matriz de %s: valor inválido en %s→%s", part.name, origin, dest)
				}
			}
		}
	}
	return nil
}

func ReadLimited(reader io.Reader, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("el archivo supera %d MB", limit/(1024*1024))
	}
	return content, nil
}
