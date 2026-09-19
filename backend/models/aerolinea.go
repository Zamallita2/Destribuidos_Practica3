package models

import (
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
	ID     uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Codigo string `json:"codigo" bson:"codigo"`
	Pais   string `json:"pais" bson:"pais"`
	Region string `json:"region" bson:"region"`
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
}

type EstadoVuelo struct {
	ID     uint   `gorm:"primaryKey" json:"id" bson:"id"`
	Nombre string `json:"nombre" bson:"nombre"`
}

type Vuelo struct {
	ID                uint      `gorm:"primaryKey" json:"id" bson:"id"`
	IDOrigen          uint      `json:"id_origen" bson:"id_origen"`
	IDDestino         uint      `json:"id_destino" bson:"id_destino"`
	IDEstadoVuelo     uint      `json:"id_estado_vuelo" bson:"id_estado_vuelo"`
	IDPuerta          uint      `json:"id_puerta" bson:"id_puerta"`
	IDAvion           uint      `json:"id_avion" bson:"id_avion"`
	LlegadaProgramada int64 `json:"llegada_programada" bson:"llegada_programada"`
	SalidaProgramada  int64 `json:"salida_programada" bson:"salida_programada"`
	LlegadaReal       int64 `json:"llegada_real" bson:"llegada_real"`
	SalidaReal        int64 `json:"salida_real" bson:"salida_real"`
	FechaLlegada      int64 `json:"fecha_llegada" bson:"fecha_llegada"`
	FechaSalida       int64 `json:"fecha_salida" bson:"fecha_salida"`
	LamportClock      int64 `json:"lamport_clock" bson:"lamport_clock"`
}

type Boleto struct {
	IDBoleto      uint    `gorm:"primaryKey" json:"id_boleto" bson:"id_boleto"`
	NombrePasajero string `json:"nombre_pasajero" bson:"nombre_pasajero"`
	EmailPasajero string `json:"email_pasajero" bson:"email_pasajero"`
	IDVuelo       uint    `json:"id_vuelo" bson:"id_vuelo"`
	IDAsiento     uint    `json:"id_asiento" bson:"id_asiento"`
	Costo         float64 `json:"costo" bson:"costo"`
	TiempoDeViaje int     `json:"tiempo_de_viaje" bson:"tiempo_de_viaje"`
	Pasaporte     string  `json:"pasaporte" bson:"pasaporte"`
	Estado        string  `json:"estado" bson:"estado"`
	LamportClock  int64   `json:"lamport_clock" bson:"lamport_clock"`
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

func (Avion) TableName() string { return "aviones" }
func (Ciudad) TableName() string { return "ciudades" }
func (Puerta) TableName() string { return "puertas" }
func (Asiento) TableName() string { return "asientos" }
func (EstadoVuelo) TableName() string { return "estados_vuelo" }
func (Vuelo) TableName() string { return "vuelos" }
func (Boleto) TableName() string { return "boletos" }
func (Precios) TableName() string { return "precios" }
func (DetallesVuelos) TableName() string { return "detalles_vuelos" }
