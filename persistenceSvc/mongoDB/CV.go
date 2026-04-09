package mongodb

import (
	"context"
	"log"
	"microservice/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoSvc struct {
	Client *mongo.Client
}

func (s *MongoSvc) InsertUser(databaseName, collectionName string, person models.Person) (*mongo.InsertOneResult, error) {
	collection := s.Client.Database(databaseName).Collection(collectionName)

	result, err := collection.InsertOne(context.Background(), person)
	if err != nil {
		return nil, err
	}
	log.Println("Inserted user with ID:", result.InsertedID)

	return result, nil
}

func (s *MongoSvc) InsertCV(databaseName, collectionName string, cv models.Profile) (*mongo.InsertOneResult, error) {
	collection := s.Client.Database(databaseName).Collection(collectionName)
	cv.Id = primitive.NewObjectID().Hex()
	cv.CreatedAt = time.Now()
	cv.UpdatedAt = time.Now()

	result, err := collection.InsertOne(context.Background(), cv)
	if err != nil {
		return nil, err
	}
	log.Println("Inserted CV with ID:", result.InsertedID)

	return result, nil
}

func (s *MongoSvc) ListAllCVs(databaseName, collectionName string) ([]models.Profile, error) {
	collection := s.Client.Database(databaseName).Collection(collectionName)

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var profiles []models.Profile

	for cursor.Next(context.Background()) {
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

func (s *MongoSvc) UpdateCV(databaseName, collectionName string, profile models.Profile) error {
	collection := s.Client.Database(databaseName).Collection(collectionName)

	_, err := collection.ReplaceOne(context.Background(), bson.M{"id": profile.Id}, profile)
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoSvc) DeleteCV(databaseName, collectionName string, id string) error {
	collection := s.Client.Database(databaseName).Collection(collectionName)

	_, err := collection.DeleteOne(context.Background(), bson.M{"id": id})
	if err != nil {
		return err
	}

	return nil
}
