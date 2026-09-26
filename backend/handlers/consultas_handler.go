package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"airres-api/db"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

type reportRow map[string]interface{}

type reportResponse struct {
	Columns []string    `json:"columns"`
	Rows    []reportRow `json:"rows"`
	Note    string      `json:"note,omitempty"`
}

// Flight IDs below one billion belong to America; the remaining IDs belong
// to Europa/Asia. Reading only the owner avoids counting replicated rows twice.
func reportNodes() ([]struct {
	conn  *gorm.DB
	where string
}, error) {
	nodes := []struct {
		conn  *gorm.DB
		where string
	}{{db.PGAmerica, "f.id < 1000000000"}, {db.PGEuropaAsia, "f.id >= 1000000000"}}
	for _, node := range nodes {
		if !db.IsAvailable(node.conn) {
			return nil, errors.New("ambas bases regionales deben estar disponibles para una consulta global")
		}
	}
	return nodes, nil
}

const passengerSQL = `WITH personas AS (
 SELECT f.id AS vuelo, f.salida_programada AS salida, o.codigo AS origen, d.codigo AS destino,
        b.nombre_pasajero AS pasajero, s.codigo AS asiento, s.clase AS clase
 FROM vuelos f JOIN boletos b ON b.id_vuelo=f.id JOIN asientos s ON s.id=b.id_asiento
 JOIN ciudades o ON o.id=f.id_origen JOIN ciudades d ON d.id=f.id_destino
 WHERE b.estado='SALED' AND %s
 UNION ALL
 SELECT f.id, f.salida_programada, o.codigo, d.codigo,
        a.item->>'passenger_name', s.codigo, s.clase
 FROM vuelos f JOIN ocupaciones_vuelo m ON m.id_vuelo=f.id
 CROSS JOIN LATERAL jsonb_array_elements(m.assignments) AS a(item)
 JOIN asientos s ON s.id=(a.item->>'seat_id')::bigint
 JOIN ciudades o ON o.id=f.id_origen JOIN ciudades d ON d.id=f.id_destino
 WHERE a.item->>'status'='SALED' AND %s
)
SELECT vuelo, salida, origen, destino, pasajero, asiento, clase FROM personas WHERE %s`

func scanReport(conn *gorm.DB, query string, args ...interface{}) ([]reportRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := conn.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]reportRow, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		item := reportRow{}
		for i, column := range columns {
			switch v := values[i].(type) {
			case []byte:
				item[column] = string(v)
			default:
				item[column] = v
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func numeric(row reportRow, key string) float64 {
	switch value := row[key].(type) {
	case int64:
		return float64(value)
	case float64:
		return value
	case string:
		n, _ := strconv.ParseFloat(value, 64)
		return n
	}
	return 0
}

func GetConsulta(c *gin.Context) {
	report := c.Param("report")
	if report == "7" {
		getConnections(c)
		return
	}
	if report != "1" && report != "2" && report != "3" && report != "4" && report != "5" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Consulta no disponible"})
		return
	}
	nodes, err := reportNodes()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(c.DefaultQuery("pasajero", "Priya Sharma"))
	if len(name) == 0 || len(name) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nombre de pasajero inválido"})
		return
	}
	year, err := strconv.Atoi(c.DefaultQuery("anio", strconv.Itoa(time.Now().Year())))
	if err != nil || year < 2000 || year > 2100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Año inválido"})
		return
	}
	var flight uint64
	if report == "2" || report == "4" {
		flight, err = strconv.ParseUint(c.Query("vuelo"), 10, 32)
		if err != nil || flight == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID de vuelo inválido"})
			return
		}
	}
	response := reportResponse{Rows: []reportRow{}}
	for _, node := range nodes {
		var query string
		var args []interface{}
		switch report {
		case "1", "2", "3":
			filter := "lower(pasajero)=lower(?)"
			args = []interface{}{name}
			if report == "2" {
				filter, args = "vuelo=?", []interface{}{flight}
			}
			query = fmt.Sprintf(passengerSQL, node.where, node.where, filter)
			if report == "1" {
				query += " AND salida < ? AND EXTRACT(YEAR FROM to_timestamp(salida) AT TIME ZONE 'UTC')=? AND vuelo IN (SELECT id FROM vuelos WHERE id_estado_vuelo IN (3,4,5,6))"
				args = append(args, time.Now().Unix(), year)
			}
			query += " ORDER BY salida DESC, vuelo, asiento"
		case "4", "5":
			filter := ""
			if report == "4" {
				filter, args = " AND f.id=?", []interface{}{flight}
			}
			query = fmt.Sprintf(`SELECT f.salida_programada AS salida, s.clase AS clase, COUNT(*) AS vendidos, SUM(b.costo) AS ingresos
 FROM vuelos f JOIN boletos b ON b.id_vuelo=f.id JOIN asientos s ON s.id=b.id_asiento
 WHERE b.estado='SALED' AND %s%s GROUP BY f.salida_programada,s.clase
 UNION ALL
 SELECT f.salida_programada, 'VIP', (SELECT COUNT(*) FROM jsonb_array_elements(m.assignments) AS a(item)
   JOIN asientos s ON s.id=(a.item->>'seat_id')::bigint WHERE a.item->>'status'='SALED' AND s.clase='VIP'), m.first_income
 FROM vuelos f JOIN ocupaciones_vuelo m ON m.id_vuelo=f.id WHERE %s%s AND m.first_income>0
 UNION ALL
 SELECT f.salida_programada, 'REGULAR', (SELECT COUNT(*) FROM jsonb_array_elements(m.assignments) AS a(item)
   JOIN asientos s ON s.id=(a.item->>'seat_id')::bigint WHERE a.item->>'status'='SALED' AND s.clase='REGULAR'), m.economy_income
 FROM vuelos f JOIN ocupaciones_vuelo m ON m.id_vuelo=f.id WHERE %s%s AND m.economy_income>0`, node.where, filter, node.where, filter, node.where, filter)
			if report == "5" {
				query = fmt.Sprintf(`SELECT f.salida_programada AS salida, s.clase AS clase, COUNT(*) AS vendidos, SUM(b.costo) AS ingresos
 FROM vuelos f JOIN boletos b ON b.id_vuelo=f.id JOIN asientos s ON s.id=b.id_asiento
 WHERE b.estado='SALED' AND %s GROUP BY f.salida_programada,s.clase
 UNION ALL SELECT f.salida_programada, 'VIP', 0, m.first_income
 FROM vuelos f JOIN ocupaciones_vuelo m ON m.id_vuelo=f.id WHERE %s AND m.first_income>0
 UNION ALL SELECT f.salida_programada, 'REGULAR', 0, m.economy_income
 FROM vuelos f JOIN ocupaciones_vuelo m ON m.id_vuelo=f.id WHERE %s AND m.economy_income>0`, node.where, node.where, node.where)
				query = `SELECT EXTRACT(EPOCH FROM date_trunc('quarter', to_timestamp(salida) AT TIME ZONE 'UTC'))::bigint AS salida,
 'ALL' AS clase, SUM(vendidos) AS vendidos, SUM(ingresos) AS ingresos FROM (` + query + `) AS sales
 GROUP BY date_trunc('quarter', to_timestamp(salida) AT TIME ZONE 'UTC')`
			}
			if report == "4" {
				args = []interface{}{flight, flight, flight}
			}
		}
		rows, err := scanReport(node.conn, query, args...)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		response.Rows = append(response.Rows, rows...)
	}
	switch report {
	case "1":
		seen := map[interface{}]bool{}
		for _, row := range response.Rows {
			seen[row["vuelo"]] = true
		}
		response.Columns = []string{"pasajero", "anio", "vuelos_realizados"}
		response.Rows = []reportRow{{"pasajero": name, "anio": year, "vuelos_realizados": len(seen)}}
	case "2":
		response.Columns = []string{"pasajero", "asiento"}
		for _, row := range response.Rows {
			delete(row, "vuelo")
			delete(row, "salida")
			delete(row, "origen")
			delete(row, "destino")
			delete(row, "clase")
		}
	case "3":
		response.Rows = oneRowPerFlight(response.Rows)
		sort.Slice(response.Rows, func(i, j int) bool {
			return numeric(response.Rows[i], "salida") > numeric(response.Rows[j], "salida")
		})
		response.Columns = []string{"vuelo", "fecha", "ruta", "asiento", "clase"}
		response.Note = "Un asiento por nombre y vuelo; los nombres repetidos en datos de prueba se consolidan."
		for _, row := range response.Rows {
			row["fecha"] = time.Unix(int64(numeric(row, "salida")), 0).UTC().Format("2006-01-02")
			row["ruta"] = fmt.Sprintf("%v - %v", row["origen"], row["destino"])
			for _, key := range []string{"salida", "origen", "destino", "pasajero"} {
				delete(row, key)
			}
			if row["clase"] == "VIP" {
				row["clase"] = "Ejecutiva"
			} else {
				row["clase"] = "Turística"
			}
		}
	case "4":
		response.Columns = []string{"clase", "asientos_vendidos", "ingresos"}
		response.Rows = aggregateClasses(response.Rows)
		response.Note = "Ingresos de boletos vendidos y del manifiesto inicial."
	case "5":
		response.Columns = []string{"trimestre", "ingresos"}
		response.Rows = aggregateQuarters(response.Rows)
		response.Note = "Incluye ventas del manifiesto inicial y boletos vendidos; excluye reservas y devoluciones."
	}
	c.JSON(http.StatusOK, response)
}

func oneRowPerFlight(rows []reportRow) []reportRow {
	seen := make(map[interface{}]bool, len(rows))
	result := make([]reportRow, 0, len(rows))
	for _, row := range rows {
		flight := row["vuelo"]
		if !seen[flight] {
			seen[flight] = true
			result = append(result, row)
		}
	}
	return result
}

func aggregateClasses(rows []reportRow) []reportRow {
	totals := map[string]reportRow{}
	for _, row := range rows {
		class := "Turística"
		if row["clase"] == "VIP" {
			class = "Ejecutiva"
		}
		entry, ok := totals[class]
		if !ok {
			entry = reportRow{"clase": class, "asientos_vendidos": 0.0, "ingresos": 0.0}
			totals[class] = entry
		}
		entry["asientos_vendidos"] = numeric(entry, "asientos_vendidos") + numeric(row, "vendidos")
		entry["ingresos"] = numeric(entry, "ingresos") + numeric(row, "ingresos")
	}
	result := []reportRow{}
	for _, class := range []string{"Ejecutiva", "Turística"} {
		if row, ok := totals[class]; ok {
			result = append(result, row)
		}
	}
	return result
}

func aggregateQuarters(rows []reportRow) []reportRow {
	totals := map[string]float64{}
	for _, row := range rows {
		date := time.Unix(int64(numeric(row, "salida")), 0).UTC()
		key := fmt.Sprintf("Q%d %d", (int(date.Month())-1)/3+1, date.Year())
		totals[key] += numeric(row, "ingresos")
	}
	keys := make([]string, 0, len(totals))
	for key := range totals {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i][3:] < keys[j][3:] || keys[i][3:] == keys[j][3:] && keys[i] < keys[j]
	})
	result := make([]reportRow, 0, len(keys))
	for _, key := range keys {
		result = append(result, reportRow{"trimestre": key, "ingresos": totals[key]})
	}
	return result
}

func getConnections(c *gin.Context) {
	response := reportResponse{Columns: []string{"servidor", "conexiones"}, Rows: []reportRow{}, Note: "Conexiones abiertas ahora; no es un contador histórico de clientes."}
	for _, node := range []struct {
		name string
		conn *gorm.DB
	}{{"PostgreSQL América", db.PGAmerica}, {"PostgreSQL Europa/Asia", db.PGEuropaAsia}} {
		var count interface{} = nil
		if db.IsAvailable(node.conn) {
			var current int64
			if err := node.conn.Raw("SELECT COUNT(*) FROM pg_stat_activity WHERE datname=current_database() AND backend_type='client backend'").Scan(&current).Error; err == nil {
				count = current
			}
		}
		response.Rows = append(response.Rows, reportRow{"servidor": node.name, "conexiones": count})
	}
	var mongoCount interface{} = nil
	if db.IsMongoAvailable() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var result bson.M
		if err := db.MongoDatabase.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result); err == nil {
			if connections, ok := result["connections"].(bson.M); ok {
				mongoCount = connections["current"]
			}
		}
	}
	response.Rows = append(response.Rows, reportRow{"servidor": "MongoDB", "conexiones": mongoCount})
	c.JSON(http.StatusOK, response)
}

var readOnlyStart = regexp.MustCompile(`(?is)^\s*(SELECT|WITH)\b`)
var forbiddenSQL = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|TRUNCATE|GRANT|REVOKE|COPY|CALL|DO|EXECUTE|SET|RESET|LOCK|VACUUM|ANALYZE|REFRESH|MERGE|INTO)\b`)

func ensureReportReader(conn *gorm.DB) error {
	if err := conn.Exec(`DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='airres_report_reader') THEN
  CREATE ROLE airres_report_reader NOLOGIN;
 END IF;
 END $$`).Error; err != nil {
		return err
	}
	return conn.Exec(`GRANT USAGE ON SCHEMA public TO airres_report_reader;
 GRANT SELECT ON aviones, ciudades, puertas, asientos, estados_vuelo, vuelos, boletos,
 precios, detalles_vuelos, ocupaciones_vuelo TO airres_report_reader`).Error
}

func RunReadOnlySQL(c *gin.Context) {
	var input struct {
		Node string `json:"node"`
		SQL  string `json:"sql"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solicitud inválida"})
		return
	}
	query := strings.TrimSpace(input.SQL)
	query = strings.TrimSuffix(query, ";")
	if len(query) == 0 || len(query) > 10000 || !readOnlyStart.MatchString(query) || forbiddenSQL.MatchString(query) || strings.Contains(query, ";") || strings.Contains(query, "--") || strings.Contains(query, "/*") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo se admite una sentencia SELECT o WITH de lectura, sin comentarios"})
		return
	}
	var conn *gorm.DB
	switch input.Node {
	case "america":
		conn = db.PGAmerica
	case "europa":
		conn = db.PGEuropaAsia
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Seleccione un nodo PostgreSQL"})
		return
	}
	if !db.IsAvailable(conn) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Nodo no disponible"})
		return
	}
	// The application connection is an owner account in the local deployment.
	// Execute user SQL with a role that can only read reporting tables.
	if err := ensureReportReader(conn); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No se pudo preparar el rol de lectura: " + err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	response := reportResponse{Rows: []reportRow{}, Note: "Máximo 500 filas. La consulta se ejecuta en el nodo elegido."}
	err := conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE airres_report_reader").Error; err != nil {
			return err
		}
		if err := tx.Exec("SET LOCAL statement_timeout = '5s'").Error; err != nil {
			return err
		}
		rows, err := tx.Raw(query).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		response.Columns, err = rows.Columns()
		if err != nil {
			return err
		}
		for rows.Next() && len(response.Rows) < 500 {
			values := make([]interface{}, len(response.Columns))
			ptrs := make([]interface{}, len(values))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			item := reportRow{}
			for i, col := range response.Columns {
				if raw, ok := values[i].([]byte); ok {
					item[col] = string(raw)
				} else {
					item[col] = values[i]
				}
			}
			response.Rows = append(response.Rows, item)
		}
		return rows.Err()
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}
