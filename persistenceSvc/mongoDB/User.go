package mongodb

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	usersDatabaseName   = "project"
	usersCollectionName = "usersdb"
)

type storedUserRole struct {
	Role string `bson:"role"`
}

// GetUserRole returns the current role stored for a user. The JWT subject is
// the hexadecimal MongoDB ObjectID issued by authenSvc.
func (s *MongoSvc) GetUserRole(ctx context.Context, userID string) (string, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	var user storedUserRole
	err = s.Client.Database(usersDatabaseName).
		Collection(usersCollectionName).
		FindOne(ctx, bson.M{"_id": objectID}).
		Decode(&user)
	if err != nil {
		log.Println("(s *MongoSvc) GetUserRole() err = ", err)
		return "", err
	}
	return user.Role, nil
}
