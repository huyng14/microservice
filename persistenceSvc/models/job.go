package models

import "time"

type Job struct {
	Id              string    `json:"id" bson:"id"`
	Title           string    `json:"title" bson:"title"`
	Company         string    `json:"company" bson:"company"`
	Location        string    `json:"location" bson:"location"`
	Type            string    `json:"type" bson:"type"`
	Description     string    `json:"description" bson:"description"`
	Requirements    []string  `json:"requirements" bson:"requirements"`
	Responsibilities []string `json:"responsibilities" bson:"responsibilities"`
	Salary          string    `json:"salary" bson:"salary"`
	CreatedAt       time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" bson:"updatedAt"`
}
