package main

import (
	"context"
	"log"
	"time"

	"airres-api/db"
	"airres-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// bootstrapMongo reconciles the data lake with both relational databases.
// Re-running it is safe: every document is replaced by its stable domain ID.
func bootstrapMongo(overwrite bool) {
	if !db.IsMongoAvailable() || !db.IsAvailable(db.PGAmerica) || !db.IsAvailable(db.PGEuropaAsia) {
		return
	}
	var planes []models.Avion
	var cities []models.Ciudad
	var gates []models.Puerta
	var states []models.EstadoVuelo
	var seats []models.Asiento
	var americanFlights, otherFlights []models.Vuelo
	var americanTickets, otherTickets []models.Boleto
	var americanOccupancy, otherOccupancy []models.OcupacionVuelo
	for _, query := range []struct {
		name string
		err  error
	}{
		{"aviones", db.PGAmerica.Find(&planes).Error},
		{"ciudades", db.PGAmerica.Find(&cities).Error},
		{"puertas", db.PGAmerica.Find(&gates).Error},
		{"estados_vuelo", db.PGAmerica.Find(&states).Error},
		{"asientos", db.PGAmerica.Find(&seats).Error},
		{"vuelos_america", db.PGAmerica.Find(&americanFlights).Error},
		{"vuelos_europa", db.PGEuropaAsia.Find(&otherFlights).Error},
		{"boletos_america", db.PGAmerica.Find(&americanTickets).Error},
		{"boletos_europa", db.PGEuropaAsia.Find(&otherTickets).Error},
		{"ocupacion_america", db.PGAmerica.Where("id_vuelo < ?", 1000000000).Find(&americanOccupancy).Error},
		{"ocupacion_europa", db.PGEuropaAsia.Where("id_vuelo >= ?", 1000000000).Find(&otherOccupancy).Error},
	} {
		if query.err != nil {
			log.Printf("[Bootstrap Mongo] Cannot read %s: %v", query.name, query.err)
			return
		}
	}
	flightByID := make(map[uint]models.Vuelo)
	for _, flight := range append(americanFlights, otherFlights...) {
		if old, ok := flightByID[flight.ID]; !ok || flight.LamportClock > old.LamportClock {
			flightByID[flight.ID] = flight
		}
	}
	ticketByID := make(map[uint]models.Boleto)
	for _, ticket := range append(americanTickets, otherTickets...) {
		if old, ok := ticketByID[ticket.IDBoleto]; !ok || ticket.LamportClock > old.LamportClock {
			ticketByID[ticket.IDBoleto] = ticket
		}
	}
	flights := make([]models.Vuelo, 0, len(flightByID))
	for _, flight := range flightByID {
		flights = append(flights, flight)
	}
	tickets := make([]models.Boleto, 0, len(ticketByID))
	for _, ticket := range ticketByID {
		tickets = append(tickets, ticket)
	}
	operations := []error{
		upsertSnapshot("aviones", planes, func(v models.Avion) bson.M { return bson.M{"id": v.ID} }, overwrite),
		upsertSnapshot("ciudades", cities, func(v models.Ciudad) bson.M { return bson.M{"id": v.ID} }, overwrite),
		upsertSnapshot("puertas", gates, func(v models.Puerta) bson.M { return bson.M{"id": v.ID} }, overwrite),
		upsertSnapshot("estados_vuelo", states, func(v models.EstadoVuelo) bson.M { return bson.M{"id": v.ID} }, overwrite),
		upsertSnapshot("asientos", seats, func(v models.Asiento) bson.M { return bson.M{"id": v.ID} }, overwrite),
		upsertSnapshot("vuelos", flights, func(v models.Vuelo) bson.M { return bson.M{"id": v.ID} }, overwrite),
		upsertSnapshot("boletos", tickets, func(v models.Boleto) bson.M { return bson.M{"id_boleto": v.IDBoleto} }, overwrite),
		upsertSnapshot("ocupaciones_vuelo", append(americanOccupancy, otherOccupancy...), func(v models.OcupacionVuelo) bson.M { return bson.M{"id_vuelo": v.IDVuelo} }, overwrite),
	}
	for _, err := range operations {
		if err != nil {
			log.Printf("[Bootstrap Mongo] Snapshot write failed: %v", err)
			return
		}
	}
	log.Printf("[Bootstrap Mongo] Reconciled %d flights and %d tickets", len(flights), len(tickets))
}

func upsertSnapshot[T any](collection string, records []T, filter func(T) bson.M, overwrite bool) error {
	coll := db.MongoDatabase.Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	key := "id"
	if collection == "boletos" {
		key = "id_boleto"
	} else if collection == "ocupaciones_vuelo" {
		key = "id_vuelo"
	}
	if _, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: key, Value: 1}}, Options: options.Index().SetUnique(true)}); err != nil {
		return err
	}
	for start := 0; start < len(records); start += 500 {
		end := start + 500
		if end > len(records) {
			end = len(records)
		}
		writes := make([]mongo.WriteModel, 0, end-start)
		for _, record := range records[start:end] {
			if !overwrite && (collection == "vuelos" || collection == "boletos" || collection == "ocupaciones_vuelo") {
				writes = append(writes, mongo.NewUpdateOneModel().SetFilter(filter(record)).SetUpdate(bson.M{"$setOnInsert": record}).SetUpsert(true))
			} else {
				writes = append(writes, mongo.NewReplaceOneModel().SetFilter(filter(record)).SetReplacement(record).SetUpsert(true))
			}
		}
		if _, err := coll.BulkWrite(ctx, writes, options.BulkWrite().SetOrdered(false)); err != nil {
			return err
		}
	}
	return nil
}
