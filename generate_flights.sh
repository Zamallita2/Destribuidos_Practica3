#!/usr/bin/env bash
# =============================================================================
#  generate_flights.sh — Genera vuelos automáticamente desde la matriz de rutas
# =============================================================================
#
#  USO:
#    ./generate_flights.sh [NUM_VUELOS] [BASE_URL] [PROB_REGRESO] [USE_API] [CSV_PATH]
#
#  ARGUMENTOS:
#    NUM_VUELOS   Número de vuelos de ida a crear          (default: 20)
#    BASE_URL     URL base del backend                     (default: http://localhost:8080)
#    PROB_REGRESO Probabilidad de vuelo de regreso (0-100) (default: 40)
#    USE_API      true  → inserta vuelos en la BD via REST API  (default: true)
#                 false → reemplaza el CSV con los vuelos generados
#    CSV_PATH     Ruta al CSV (solo si USE_API=false)
#                 (default: dataset/02 - Practica 3 Dataset Flights.csv)
#
#  DISTRIBUCIÓN DE FECHAS (septiembre 2026):
#    ≥60% de los vuelos parten del 26 de septiembre en adelante.
#    El resto parte desde ahora hasta el 25 de septiembre (fin de día).
#
#  EJEMPLOS:
#    ./generate_flights.sh                              # 20 vuelos via API
#    ./generate_flights.sh 50                           # 50 vuelos via API
#    ./generate_flights.sh 30 http://localhost:8080 40 true   # API explícito
#    ./generate_flights.sh 200 http://localhost:8080 50 false  # reemplaza CSV
#    ./generate_flights.sh 100 http://localhost:8080 40 false "dataset/mi_archivo.csv"
#
#  REQUISITOS: curl, jq
# =============================================================================

set -euo pipefail

# ── Parámetros ────────────────────────────────────────────────────────────────
NUM_VUELOS="${1:-20}"
BASE_URL="${2:-http://localhost:8080}"
PROB_REGRESO="${3:-40}"
USE_API="${4:-true}"

# Ruta al CSV relativa al directorio del script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_CSV="$SCRIPT_DIR/dataset/02 - Practica 3 Dataset Flights.csv"
CSV_PATH="${5:-$DEFAULT_CSV}"

NOW=$(date +%s)

# ── Ventanas de fechas — septiembre 2026 ──────────────────────────────────────
WIN_A_START=$(date -d "2026-09-26 00:00:00 UTC" +%s)
WIN_A_END=$(date   -d "2026-09-30 23:59:59 UTC" +%s)
WIN_B_START=$NOW
WIN_B_END=$(date   -d "2026-09-25 23:59:59 UTC" +%s)

# Si ya pasamos la ventana B todo va a la ventana A
if [[ "$WIN_B_START" -ge "$WIN_B_END" ]]; then
  WIN_B_AVAILABLE=0
else
  WIN_B_AVAILABLE=1
fi

PROB_WIN_A=60   # ≥60% de vuelos caen en Sep 26-30

# ── Colores ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[✓]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[!]${NC}    $*"; }
err()     { echo -e "${RED}[✗]${NC}    $*" >&2; }
header()  { echo -e "\n${BOLD}${CYAN}$*${NC}\n"; }

# ── Validar dependencias ──────────────────────────────────────────────────────
for cmd in curl jq; do
  if ! command -v "$cmd" &>/dev/null; then
    err "Comando requerido '$cmd' no encontrado. Instálalo e intenta de nuevo."
    exit 1
  fi
done

# ── Encabezado ────────────────────────────────────────────────────────────────
if [[ "$USE_API" == "false" ]]; then
  header "✈  Generador de Vuelos — AirRes  [MODO CSV]"
  info "Destino      : $CSV_PATH"
else
  header "✈  Generador de Vuelos — AirRes  [MODO API]"
  info "Backend      : $BASE_URL"
fi
info "Vuelos de ida: $NUM_VUELOS"
info "Ida+vuelta   : ${PROB_REGRESO}% probabilidad"
info "Sep 26-30    : ≥${PROB_WIN_A}% de los vuelos"
[[ "$WIN_B_AVAILABLE" -eq 1 ]] && \
  info "Sep 25 (hoy) : ≤$(( 100 - PROB_WIN_A ))% de los vuelos"
echo ""

# ── Conectar al backend ───────────────────────────────────────────────────────
info "Obteniendo datos del backend…"

fetch() {
  local endpoint="$1"
  local result
  if ! result=$(curl -sf --max-time 10 "$BASE_URL$endpoint"); then
    err "No se pudo conectar a $BASE_URL$endpoint"
    err "¿Está corriendo el contenedor Docker? (docker compose up -d)"
    exit 1
  fi
  echo "$result"
}

CITIES=$(fetch "/api/ciudades")
TIEMPOS=$(fetch "/api/tiempos")
PRECIOS=$(fetch "/api/precios")
AVIONES=$(fetch "/api/aviones")
PUERTAS=$(fetch "/api/puertas")

NUM_CITIES=$(echo "$CITIES" | jq 'length')
NUM_PLANES=$(echo "$AVIONES" | jq 'length')

if [[ "$NUM_CITIES" -eq 0 ]]; then
  err "No se encontraron ciudades. ¿Está la base de datos inicializada?"
  exit 1
fi
if [[ "$NUM_PLANES" -eq 0 ]]; then
  err "No se encontraron aeronaves. ¿Está la base de datos inicializada?"
  exit 1
fi

info "Ciudades encontradas : $NUM_CITIES"
info "Aeronaves encontradas: $NUM_PLANES"

# ── Mapa código_ciudad → id  y  id → código ──────────────────────────────────
CITY_MAP=$(echo "$CITIES" | jq '[.[] | {key: .codigo, value: .id}]    | from_entries')
CITY_CODE=$(echo "$CITIES" | jq '[.[] | {key: (.id|tostring), value: .codigo}] | from_entries')

# ── Mapa id_ciudad → nombre_de_puerta (primera puerta disponible) ─────────────
# Formato de la API: [{id, puerta, id_ciudad}, ...]
GATE_MAP=$(echo "$PUERTAS" | jq '
  reduce .[] as $p ({};
    if .[$p.id_ciudad | tostring] == null
    then .[$p.id_ciudad | tostring] = $p.puerta
    else .
    end
  )
')

# ── Matrices de tarifas ───────────────────────────────────────────────────────
ECO_FARES=$(echo "$PRECIOS" | jq '.matriz_precios_regular // {}')
VIP_FARES=$(echo "$PRECIOS" | jq '.matriz_precios_vip // {}')

# ── Rutas válidas ─────────────────────────────────────────────────────────────
ROUTES=$(jq -n \
  --argjson times "$TIEMPOS" \
  --argjson eco   "$ECO_FARES" \
  --argjson vip   "$VIP_FARES" '
  [
    $times | to_entries[] |
    .key as $from |
    .value | to_entries[] |
    select(.key != $from and .value > 0) |
    .key as $to |
    .value as $hours |
    select(
      ( $eco[$from] != null and $eco[$from][$to] != null
        and ($eco[$from][$to] | type) == "number" and $eco[$from][$to] > 0 ) or
      ( $vip[$from] != null and $vip[$from][$to] != null
        and ($vip[$from][$to] | type) == "number" and $vip[$from][$to] > 0 )
    ) |
    { from: $from, to: $to, hours: $hours }
  ]
')

NUM_ROUTES=$(echo "$ROUTES" | jq 'length')
if [[ "$NUM_ROUTES" -eq 0 ]]; then
  err "No se encontraron rutas válidas en la matriz de tiempos/tarifas."
  exit 1
fi
info "Rutas válidas encontradas: $NUM_ROUTES"

# ── Array de IDs de aeronaves ─────────────────────────────────────────────────
PLANE_IDS=$(echo "$AVIONES" | jq '[.[].id]')

# ── Funciones utilitarias ─────────────────────────────────────────────────────

rand_int() {
  local n="$1"
  echo $(( $(od -An -N4 -tu4 /dev/urandom | tr -d ' \n') % n ))
}

rand_ts_in_range() {
  local start="$1" end="$2"
  local span=$(( end - start ))
  [[ "$span" -le 0 ]] && { echo "$start"; return; }
  echo $(( start + $(rand_int "$span") ))
}

# Devuelve un Unix timestamp dentro de la distribución Sep 2026
pick_departure() {
  if [[ $(rand_int 100) -lt "$PROB_WIN_A" || "$WIN_B_AVAILABLE" -eq 0 ]]; then
    rand_ts_in_range "$WIN_A_START" "$WIN_A_END"
  else
    rand_ts_in_range "$WIN_B_START" "$WIN_B_END"
  fi
}

# Convierte Unix timestamp a formato CSV:  MM/DD/YY,HH:MM
ts_to_csv_date_time() {
  local ts="$1"
  date -d "@$ts" '+%m/%d/%y,%H:%M' 2>/dev/null || echo "00/00/00,00:00"
}

# Obtiene la puerta de la ciudad origen (o genera una genérica)
get_gate() {
  local city_id="$1"
  local gate
  gate=$(echo "$GATE_MAP" | jq -r --arg id "$city_id" '.[$id] // empty')
  [[ -z "$gate" ]] && gate="G$(printf '%02d' $(( $(rand_int 30) + 1 )))"
  echo "$gate"
}

# ── MODO API: crear vuelo via REST ────────────────────────────────────────────
create_flight_api() {
  local origin_code="$1" dest_code="$2" plane_id="$3" depart_unix="$4"

  local origin_id dest_id
  origin_id=$(echo "$CITY_MAP" | jq --arg c "$origin_code" '.[$c] // empty')
  dest_id=$(echo "$CITY_MAP"   | jq --arg c "$dest_code"   '.[$c] // empty')

  if [[ -z "$origin_id" || -z "$dest_id" ]]; then
    warn "Código de ciudad desconocido ($origin_code o $dest_code), omitiendo."
    return 1
  fi
  if [[ "$depart_unix" -le "$NOW" ]]; then
    warn "Timestamp en el pasado para $origin_code → $dest_code, omitiendo."
    return 1
  fi

  local payload
  payload=$(jq -n \
    --argjson orig  "$origin_id" \
    --argjson dest  "$dest_id" \
    --argjson plane "$plane_id" \
    --argjson dep   "$depart_unix" \
    '{id_origen: $orig, id_destino: $dest, id_avion: $plane, salida_programada: $dep}')

  local tmp_body http_code body flight_id depart_str
  tmp_body=$(mktemp)
  http_code=$(curl -s -o "$tmp_body" -w "%{http_code}" \
    -X POST -H "Content-Type: application/json" \
    -d "$payload" --max-time 15 "$BASE_URL/api/vuelos" 2>/dev/null || echo "000")
  body=$(cat "$tmp_body"); rm -f "$tmp_body"

  if [[ "$http_code" == "200" || "$http_code" == "201" ]]; then
    flight_id=$(echo "$body" | jq '.id // "?"' 2>/dev/null || echo "?")
    depart_str=$(date -d "@$depart_unix" '+%d/%m/%Y %H:%M UTC' 2>/dev/null || echo "@$depart_unix")
    success "Vuelo #${flight_id}: ${origin_code} → ${dest_code} | Aeronave ${plane_id} | ${depart_str}"
    return 0
  else
    local err_msg
    err_msg=$(echo "$body" | jq -r '.error // empty' 2>/dev/null || true)
    [[ -z "$err_msg" ]] && err_msg="HTTP $http_code"
    warn "Error ${origin_code} → ${dest_code}: ${err_msg}"
    return 1
  fi
}

# ── MODO CSV: devuelve una línea CSV para agregar al buffer ───────────────────
# Salida: "MM/DD/YY,HH:MM,ORIGIN,DEST,AIRCRAFT_ID,SCHEDULED,GATE"
build_csv_row() {
  local origin_code="$1" dest_code="$2" plane_id="$3" depart_unix="$4"

  if [[ "$depart_unix" -le "$NOW" ]]; then
    return 1
  fi

  local date_time gate origin_id
  date_time=$(ts_to_csv_date_time "$depart_unix")
  origin_id=$(echo "$CITY_MAP" | jq --arg c "$origin_code" '.[$c] // empty')
  gate=$(get_gate "$origin_id")

  echo "${date_time},${origin_code},${dest_code},${plane_id},SCHEDULED,${gate}"
}

# ── Bucle principal ───────────────────────────────────────────────────────────
header "Iniciando generación de vuelos…"

CREATED=0
FAILED=0
ATTEMPT=0
MAX_ATTEMPTS=$(( NUM_VUELOS * 8 ))

# Buffer de líneas CSV (solo usado en MODO CSV)
CSV_ROWS=()

while [[ "$CREATED" -lt "$NUM_VUELOS" && "$ATTEMPT" -lt "$MAX_ATTEMPTS" ]]; do
  ATTEMPT=$(( ATTEMPT + 1 ))

  # Selección aleatoria de ruta y aeronave
  ROUTE=$(echo "$ROUTES" | jq ".[ $(rand_int "$NUM_ROUTES") ]")
  FROM=$(echo  "$ROUTE"  | jq -r '.from')
  TO=$(echo    "$ROUTE"  | jq -r '.to')
  HOURS=$(echo "$ROUTE"  | jq    '.hours')
  PLANE_ID=$(echo "$PLANE_IDS" | jq ".[ $(rand_int "$NUM_PLANES") ]")

  DEPART=$(pick_departure)

  # ─── Modo API ──────────────────────────────────────────────────────────────
  if [[ "$USE_API" == "true" ]]; then
    if create_flight_api "$FROM" "$TO" "$PLANE_ID" "$DEPART"; then
      CREATED=$(( CREATED + 1 ))

      # Vuelo de regreso opcional
      if [[ $(rand_int 100) -lt "$PROB_REGRESO" ]]; then
        DURATION_SECS=$(echo "$HOURS" | jq 'floor * 3600 | floor')
        LAYOVER_SECS=$(( ($(rand_int 5) + 2) * 3600 ))
        RETURN_DEPART=$(( DEPART + DURATION_SECS + LAYOVER_SECS ))

        if [[ "$RETURN_DEPART" -le "$WIN_A_END" ]]; then
          RET_PLANE_ID=$(echo "$PLANE_IDS" | jq ".[ $(rand_int "$NUM_PLANES") ]")
          create_flight_api "$TO" "$FROM" "$RET_PLANE_ID" "$RETURN_DEPART" || true
        fi
      fi
    else
      FAILED=$(( FAILED + 1 ))
    fi

  # ─── Modo CSV ──────────────────────────────────────────────────────────────
  else
    local_row=""
    if local_row=$(build_csv_row "$FROM" "$TO" "$PLANE_ID" "$DEPART" 2>/dev/null); then
      depart_str=$(date -d "@$DEPART" '+%d/%m/%Y %H:%M' 2>/dev/null || echo "@$DEPART")
      success "Fila CSV: ${FROM} → ${TO} | Aeronave ${PLANE_ID} | ${depart_str}"
      CSV_ROWS+=("$local_row")
      CREATED=$(( CREATED + 1 ))

      # Vuelo de regreso opcional
      if [[ $(rand_int 100) -lt "$PROB_REGRESO" ]]; then
        DURATION_SECS=$(echo "$HOURS" | jq 'floor * 3600 | floor')
        LAYOVER_SECS=$(( ($(rand_int 5) + 2) * 3600 ))
        RETURN_DEPART=$(( DEPART + DURATION_SECS + LAYOVER_SECS ))

        if [[ "$RETURN_DEPART" -le "$WIN_A_END" ]]; then
          RET_PLANE_ID=$(echo "$PLANE_IDS" | jq ".[ $(rand_int "$NUM_PLANES") ]")
          ret_row=""
          if ret_row=$(build_csv_row "$TO" "$FROM" "$RET_PLANE_ID" "$RETURN_DEPART" 2>/dev/null); then
            CSV_ROWS+=("$ret_row")
          fi
        fi
      fi
    else
      FAILED=$(( FAILED + 1 ))
    fi
  fi
done

# ── Escribir CSV si aplica ────────────────────────────────────────────────────
if [[ "$USE_API" == "false" ]]; then
  echo ""
  info "Escribiendo ${#CSV_ROWS[@]} filas en: $CSV_PATH"

  # Verificar que el directorio existe
  CSV_DIR="$(dirname "$CSV_PATH")"
  if [[ ! -d "$CSV_DIR" ]]; then
    err "El directorio '$CSV_DIR' no existe."
    exit 1
  fi

  # Escribir cabecera + nuevas filas
  {
    echo "flight_date,flight_time,origin,destination,aircraft_id,status,gate"
    for row in "${CSV_ROWS[@]}"; do
      echo "$row"
    done
  } > "$CSV_PATH"

  success "CSV actualizado con ${#CSV_ROWS[@]} vuelos → $CSV_PATH"
fi

# ── Resumen ───────────────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [[ "$CREATED" -ge "$NUM_VUELOS" ]]; then
  success "${BOLD}¡Listo! Procesados ${CREATED} vuelo(s) de ida. Fallidos: ${FAILED}.${NC}"
else
  warn "Se procesaron $CREATED de $NUM_VUELOS vuelos. Fallidos: $FAILED."
fi
if [[ "$USE_API" == "true" ]]; then
  info "Ocupación de asientos generada automáticamente por el backend."
else
  info "Recarga el CSV en el backend con el endpoint de importación."
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
