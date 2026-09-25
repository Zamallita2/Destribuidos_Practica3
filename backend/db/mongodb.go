package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var MongoDatabase *mongo.Database

func InitMongoDB() {
	uri := os.Getenv("DB_MONGO_URI")
	if uri == "" {
		uri = "mongodb://root:root@localhost:27017/"
	}
	dbName := os.Getenv("DB_MONGO_NAME")
	if dbName == "" {
		dbName = "airres_sync"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Failed to create mongo client:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Println("MongoDB unavailable at startup; waiting for recovery:", err)
	}

	MongoClient = client
	MongoDatabase = client.Database(dbName)
	log.Println("Connected to MongoDB")
}

func IsMongoAvailable() bool {
	if MongoClient == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	return MongoClient.Ping(ctx, nil) == nil
}
