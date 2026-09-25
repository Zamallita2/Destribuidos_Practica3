package main

import (
	"context"
	"log"
	"time"

	"airres-api/db"
	"go.mongodb.org/mongo-driver/bson"
)

// pruneMongoOrphans runs before the HTTP server accepts requests. It removes
// documents left by the former import that duplicated regional flight IDs.
func pruneMongoOrphans() {
	if !db.IsMongoAvailable() || !db.IsAvailable(db.PGAmerica) || !db.IsAvailable(db.PGEuropaAsia) {
		return
	}
	var american, other []uint
	if db.PGAmerica.Table("vuelos").Where("id < ?", 1000000000).Pluck("id", &american).Error != nil {
		return
	}
	if db.PGEuropaAsia.Table("vuelos").Where("id >= ?", 1000000000).Pluck("id", &other).Error != nil {
		return
	}
	ids := append(american, other...)
	if len(ids) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	result, err := db.MongoDatabase.Collection("vuelos").DeleteMany(ctx, bson.M{"id": bson.M{"$nin": ids}})
	if err != nil {
		log.Printf("[Mongo Cleanup] stale flights removal failed: %v", err)
		return
	}
	log.Printf("[Mongo Cleanup] removed %d obsolete flight documents", result.DeletedCount)
	if _, err := db.MongoDatabase.Collection("ocupaciones_vuelo").DeleteMany(ctx, bson.M{"id_vuelo": bson.M{"$nin": ids}}); err != nil {
		log.Printf("[Mongo Cleanup] stale manifests removal failed: %v", err)
	}
}
