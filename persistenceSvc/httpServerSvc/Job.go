package httpServerSvc

import (
	"fmt"
	"microservice/authorization"
	"microservice/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const jobDatabaseName = "project"
const jobCollectionName = "jobs"

func (s *HttpSvc) HandleListJobs(c *gin.Context) {
	user, _ := authorization.User(c)
	jobs, err := s.Service.ListJobs(user)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, jobs)
}

func (s *HttpSvc) HandleDeleteJob(c *gin.Context) {
	user, _ := authorization.User(c)
	id := c.Param("id")
	err := s.Service.DeleteJob(user, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"message": "Job deleted successfully"})
}

func (s *HttpSvc) HandleCreateJob(c *gin.Context) {
	user, _ := authorization.User(c)
	var job models.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	job.Id = primitive.NewObjectID().Hex()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	result, err := s.Service.CreateJob(user, job)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(201, gin.H{
		"message": "Job created successfully",
		"id":      fmt.Sprint(result.InsertedID),
	})
}

func (s *HttpSvc) HandleUpdateJob(c *gin.Context) {
	user, _ := authorization.User(c)
	var job models.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	job.Id = c.Param("id")

	err := s.Service.UpdateJob(user, job)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"message": "Job updated successfully"})
}
