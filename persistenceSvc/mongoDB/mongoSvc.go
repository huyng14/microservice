package mongodb

import (
	"context"
	"log"
	"microservice/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoSvc struct {
	Client *mongo.Client
}

func (s *MongoSvc) InsertUser(databaseName, collectionName string, person models.Person) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := s.Client.Database(databaseName).Collection(collectionName)

	result, err := collection.InsertOne(ctx, person)
	if err != nil {
		return nil, err
	}
	log.Println("Inserted user with ID:", result.InsertedID)

	return result, nil
}

func (s *MongoSvc) InsertCV(databaseName, collectionName string, cv models.Profile) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := s.Client.Database(databaseName).Collection(collectionName)
	cv.CreatedAt = time.Now()
	cv.UpdatedAt = time.Now()

	result, err := collection.InsertOne(ctx, cv)
	if err != nil {
		return nil, err
	}
	log.Println("Inserted CV with ID:", result.InsertedID)

	return result, nil
}

func (s *MongoSvc) ListAllCVs(databaseName, collectionName string) ([]models.Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := s.Client.Database(databaseName).Collection(collectionName)

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var profiles []models.Profile
	// if err = cursor.All(ctx, &profiles); err != nil {
	// 	return nil, err
	// }

	for cursor.Next(ctx) {
		var profile models.Profile
		if err := cursor.Decode(&profile); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}
