package httpServerSvc

import (
	"fmt"
	"microservice/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	job.Id = primitive.NewObjectID().Hex()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
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

func (s *HttpSvc) HandleUpdateJob(c *gin.Context) {
	var job models.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	job.Id = c.Param("id")

	err := s.MongoSvc.UpdateJob(jobDatabaseName, jobCollectionName, job)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"message": "Job updated successfully"})
}
