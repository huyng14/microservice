package models

import "time"

type User struct {
	ID        uint64    `bson:"id,omitempty" json:"id"`
	Name      string    `bson:"name" json:"name"`
	Email     string    `bson:"email" json:"email"`
	CreatedAt time.Time `bson:"created_at,omitempty" json:"created_at"`
}
