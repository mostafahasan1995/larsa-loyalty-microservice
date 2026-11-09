package database

import (
	"context"
	"log"
	"loyalty/pkg/models"
	"loyalty/pkg/util"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SeedConfig() {

	DB_NAME := util.GetEnv("DB_NAME", "WgcDev")
	collection := Mdb.Database(DB_NAME).Collection("loyaltySettings")

	// Create a unique index on the 'key' field
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"key": 1}, // Index in ascending order on the 'key' field
		Options: options.Index().SetUnique(true),
	}

	if _, err := collection.Indexes().CreateOne(context.TODO(), indexModel); err != nil {
		log.Fatal(err)
	}

	// Check if the data already exists
	var result bson.M
	err := collection.FindOne(context.TODO(), bson.M{"seeded": true}).Decode(&result)
	if err == nil {
		log.Println("Seed data already exists. Skipping seeding.")
		return
	} else if err != mongo.ErrNoDocuments {
		log.Fatal(err)
	}

	// Seed data
	// Add any number of keys
	seedData := []models.Settings{
		{
			Key:     "DEFAULT_WALLET_POINTS",
			Value:   "100",
			Visible: true,
		},
		{
			Key:     "DEFAULT_EXPERIENCE_VALUE",
			Value:   "50",
			Visible: true,
		},
		{
			Key:     "DEFAULT_FEEDBACK_VALUE",
			Value:   "50",
			Visible: true,
		},
	}

	seedDataInterface := make([]interface{}, len(seedData))
	for i, v := range seedData {
		seedDataInterface[i] = v
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := collection.InsertMany(ctx, seedDataInterface)
	if err != nil {
		log.Fatal(err)
	}

	// Insert the flag to indicate seeding is done
	_, err = collection.InsertOne(ctx, bson.M{"seeded": true, "timestamp": time.Now()})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Inserted documents: ", res.InsertedIDs)
}
