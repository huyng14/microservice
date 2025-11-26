package models

import "time"

type Person struct {
	ID         uint64    `bson:"id,omitempty" json:"id"`
	Name       string    `bson:"name" json:"name"`
	Email      string    `bson:"email,omitempty" json:"email"`
	Experience string    `bson:"experience,omitempty" json:"experience"`
	Skills     string    `bson:"skills,omitempty" json:"skills"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}
