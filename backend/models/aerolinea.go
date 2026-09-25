package models

import (
	"time"

	"gorm.io/datatypes"
)

type Avion struct {
	ID              uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Nombre          string `json:"nombre" bson:"nombre"`
	AsientosRegular int    `json:"asientos_regular" bson:"asientos_regular"`
	AsientosVip     int    `json:"asientos_vip" bson:"asientos_vip"`
	Fabricante      string `json:"fabricante" bson:"fabricante"`
}

type Ciudad struct {
	ID       uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Codigo   string `json:"codigo" bson:"codigo"`
	Pais     string `json:"pais" bson:"pais"`
	Region   string `json:"region" bson:"region"`
	TimeZone string `json:"time_zone" bson:"time_zone"`
}

type Puerta struct {
	ID       uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Puerta   string `json:"puerta" bson:"puerta"`
	IDCiudad uint   `json:"id_ciudad" bson:"id_ciudad"`
}

type Asiento struct {
	ID           uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Codigo       string `json:"codigo" bson:"codigo"`
	IDAvion      uint   `json:"id_avion" bson:"id_avion"`
	Estado       string `json:"estado" bson:"estado"`
	Clase        string `json:"clase" bson:"clase"`
	LamportClock int64  `json:"lamport_clock" bson:"lamport_clock"`
	VectorClock  string `json:"vector_clock" bson:"vector_clock"`
	SourceNode   string `json:"source_node" bson:"source_node"`
}

type EstadoVuelo struct {
	ID     uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Nombre string `json:"nombre" bson:"nombre"`
}

type Vuelo struct {
	ID                uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Demo              bool   `json:"demo" bson:"demo"`
	IDOrigen          uint   `json:"id_origen" bson:"id_origen"`
	IDDestino         uint   `json:"id_destino" bson:"id_destino"`
	IDEstadoVuelo     uint   `json:"id_estado_vuelo" bson:"id_estado_vuelo"`
	IDPuerta          uint   `json:"id_puerta" bson:"id_puerta"`
	IDAvion           uint   `json:"id_avion" bson:"id_avion"`
	LlegadaProgramada int64  `json:"llegada_programada" bson:"llegada_programada"`
	SalidaProgramada  int64  `json:"salida_programada" bson:"salida_programada"`
	LlegadaReal       int64  `json:"llegada_real" bson:"llegada_real"`
	SalidaReal        int64  `json:"salida_real" bson:"salida_real"`
	FechaLlegada      int64  `json:"fecha_llegada" bson:"fecha_llegada"`
	FechaSalida       int64  `json:"fecha_salida" bson:"fecha_salida"`
	LamportClock      int64  `json:"lamport_clock" bson:"lamport_clock"`
	VectorClock       string `json:"vector_clock" bson:"vector_clock"`
	SourceNode        string `json:"source_node" bson:"source_node"`
}

type Boleto struct {
	IDBoleto         uint    `gorm:"primaryKey" json:"id_boleto" bson:"id_boleto"`
	NombrePasajero   string  `json:"nombre_pasajero" bson:"nombre_pasajero"`
	EmailPasajero    string  `json:"email_pasajero" bson:"email_pasajero"`
	IDVuelo          uint    `json:"id_vuelo" bson:"id_vuelo"`
	IDAsiento        uint    `json:"id_asiento" bson:"id_asiento"`
	Clase            string  `json:"clase" bson:"clase"`
	Costo            float64 `json:"costo" bson:"costo"`
	TiempoDeViaje    int     `json:"tiempo_de_viaje" bson:"tiempo_de_viaje"`
	Pasaporte        string  `json:"pasaporte" bson:"pasaporte"`
	PurchaseTimeZone string  `json:"time_zone_compra" bson:"time_zone_compra"`
	Estado           string  `json:"estado" bson:"estado"`
	AvailableAt      int64   `json:"available_at" bson:"available_at"`
	LamportClock     int64   `json:"lamport_clock" bson:"lamport_clock"`
	VectorClock      string  `json:"vector_clock" bson:"vector_clock"`
	SourceNode       string  `json:"source_node" bson:"source_node"`
}

type Precios struct {
	ID                   uint           `gorm:"primaryKey" json:"id" bson:"id"`
	MatrizPreciosRegular datatypes.JSON `json:"matriz_precios_regular" bson:"matriz_precios_regular"`
	MatrizPreciosVip     datatypes.JSON `json:"matriz_precios_vip" bson:"matriz_precios_vip"`
}

type DetallesVuelos struct {
	ID            uint           `gorm:"primaryKey" json:"id" bson:"id"`
	MatrizTiempos datatypes.JSON `json:"matriz_tiempos" bson:"matriz_tiempos"`
}

type SyncOutbox struct {
	EventID       string         `gorm:"primaryKey;size:36"`
	Action        string         `gorm:"size:10"`
	Entity        string         `gorm:"size:30"`
	Payload       datatypes.JSON `gorm:"type:jsonb"`
	LamportClock  int64
	VectorClock   string
	NodeID        string
	DeliveredAt   int64 `gorm:"index"`
	NextAttemptAt int64 `gorm:"index"`
	Attempts      int
	LastError     string
	CreatedAt     time.Time
}

// OcupacionVuelo stores the initial passenger manifest for one flight.
// Seats not present in Assignments remain available for new purchases.
type OcupacionVuelo struct {
	IDVuelo       uint                   `gorm:"primaryKey" json:"id_vuelo" bson:"id_vuelo"`
	MatrixHash    string                 `gorm:"index" json:"matrix_hash" bson:"matrix_hash"`
	Assignments   []FlightSeatAssignment `gorm:"type:jsonb;serializer:json" json:"assignments" bson:"assignments"`
	EligibleSeats int                    `json:"eligible_seats" bson:"eligible_seats"`
	SoldCount     int                    `json:"sold_count" bson:"sold_count"`
	ReservedCount int                    `json:"reserved_count" bson:"reserved_count"`
	FirstIncome   float64                `json:"first_income" bson:"first_income"`
	EconomyIncome float64                `json:"economy_income" bson:"economy_income"`
	LamportClock  int64                  `json:"lamport_clock" bson:"lamport_clock"`
	VectorClock   string                 `json:"vector_clock" bson:"vector_clock"`
	SourceNode    string                 `json:"source_node" bson:"source_node"`
}

type FlightSeatAssignment struct {
	SeatID         uint   `json:"seat_id" bson:"seat_id"`
	Status         string `json:"status" bson:"status"`
	PassengerName  string `json:"passenger_name" bson:"passenger_name"`
	PassengerEmail string `json:"passenger_email" bson:"passenger_email"`
	Passport       string `json:"passport" bson:"passport"`
}

type MigrationMarker struct {
	Key string `gorm:"primaryKey;size:80"`
}

type IDAllocator struct {
	Name      string `gorm:"primaryKey;size:40"`
	NextValue uint
}

func (Avion) TableName() string           { return "aviones" }
func (Ciudad) TableName() string          { return "ciudades" }
func (Puerta) TableName() string          { return "puertas" }
func (Asiento) TableName() string         { return "asientos" }
func (EstadoVuelo) TableName() string     { return "estados_vuelo" }
func (Vuelo) TableName() string           { return "vuelos" }
func (Boleto) TableName() string          { return "boletos" }
func (Precios) TableName() string         { return "precios" }
func (DetallesVuelos) TableName() string  { return "detalles_vuelos" }
func (SyncOutbox) TableName() string      { return "sync_outbox" }
func (OcupacionVuelo) TableName() string  { return "ocupaciones_vuelo" }
func (MigrationMarker) TableName() string { return "migration_markers" }
func (IDAllocator) TableName() string     { return "id_allocators" }
