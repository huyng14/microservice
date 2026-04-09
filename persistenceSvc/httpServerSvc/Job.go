package httpServerSvc

import (
	"fmt"
	"microservice/models"
	"time"

	"github.com/gin-gonic/gin"
)

const jobDatabaseName = "project"
const jobCollectionName = "jobs"

func (s *HttpSvc) HandleListJobs(c *gin.Context) {
	jobs := []models.Job{}
	jobs, err := s.MongoSvc.ListAllJobs(jobDatabaseName, jobCollectionName)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, jobs)
}

func (s *HttpSvc) HandleDeleteJob(c *gin.Context) {
	id := c.Param("id")
	err := s.MongoSvc.DeleteJob(jobDatabaseName, jobCollectionName, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"message": "Job deleted successfully"})
}

func (s *HttpSvc) HandleCreateJob(c *gin.Context) {
	var job models.Job
	// if err := c.ShouldBindJSON(&job); err != nil {
	// 	c.JSON(400, gin.H{"error": err.Error()})
	// 	return
	// }
	job = models.Job{
		Id:               "1",
		Title:            "Senior React Developer",
		Company:          "Innovation Labs",
		Location:         "San Francisco, CA (Remote)",
		Type:             "Full-time",
		Description:      "We are looking for an experienced React developer to join our team and help build the next generation of our product.",
		Requirements:     []string{"5+ years React experience", "TypeScript", "Node.js", "AWS", "GraphQL", "Team leadership"},
		Responsibilities: []string{"Lead frontend development", "Mentor junior developers", "Architecture decisions", "Code reviews"},
		Salary:           "$140,000 - $180,000",
		CreatedAt:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
	}

	result, err := s.MongoSvc.InsertJob(jobDatabaseName, jobCollectionName, job)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"message": "Job created successfully",
		"id":      fmt.Sprint(result.InsertedID),
	})
}
