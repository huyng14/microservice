package mongodb

import (
	"context"
	"fmt"
	"log"
	"microservice/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *MongoSvc) InsertJob(databaseName, collectionName string, job models.Job) (*mongo.InsertOneResult, error) {
	collection := s.Client.Database(databaseName).Collection(collectionName)
	job.Id = primitive.NewObjectID().Hex()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	result, err := collection.InsertOne(context.Background(), job)
	if err != nil {
		return nil, err
	}
	log.Println("Inserted Job with ID:", result.InsertedID)

	return result, nil
}

func (s *MongoSvc) ListAllJobs(databaseName, collectionName string) ([]models.Job, error) {
	collection := s.Client.Database(databaseName).Collection(collectionName)

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var jobs []models.Job
	for cursor.Next(context.Background()) {
		var job models.Job
		if err := cursor.Decode(&job); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *MongoSvc) DeleteJob(databaseName, collectionName, id string) error {
	collection := s.Client.Database(databaseName).Collection(collectionName)

	result, err := collection.DeleteOne(context.Background(), bson.M{"id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("job with ID %s not found", id)
	}
	return nil
}
