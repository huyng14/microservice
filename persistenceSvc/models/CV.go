package models

import "time"

type Experience struct {
	Id          string `json:"id"`
	Company     string `json:"company"`
	Position    string `json:"position"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Description string `json:"description"`
}

type Education struct {
	Id             string `json:"id"`
	Institution    string `json:"institution"`
	Degree         string `json:"degree"`
	Field          string `json:"field"`
	GraduationDate string `json:"graduationDate"`
}

type Profile struct {
	Id         string       `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Phone      string       `json:"phone"`
	Title      string       `json:"title"`
	Summary    string       `json:"summary"`
	Skills     []string     `json:"skills"`
	Experience []Experience `json:"experience"`
	Education  []Education  `json:"education"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}
