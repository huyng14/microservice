package httpServerSvc

import (
	"errors"
	"microservice/authorization"
	clientSide "microservice/clientSide"
	"microservice/models"
	"microservice/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HttpSvc struct {
	Service   *services.PersistenceService
	UserRoles UserRoleStore
}

const databaseName = "project"
const collectionName = "consultants"

// func (s *HttpSvc) HandleGetPerson(c *gin.Context) {
// 	id := c.Param("id")
// 	var profile models.Profile

// 	if id == "1" {
// 		profile = models.Profile{
// 			Id:      "1",
// 			Name:    "Sarah Johnson",
// 			Email:   "sarah.johnson@email.com",
// 			Phone:   "+1 (555) 123-4567",
// 			Title:   "Senior Full Stack Developer",
// 			Summary: "Experienced software engineer with 8+ years in web development, specializing in React, Node.js, and cloud technologies.",
// 			Skills:  []string{"React", "Node.js", "TypeScript", "AWS", "PostgreSQL", "Docker", "GraphQL", "REST APIs"},
// 			Experience: []models.Experience{
// 				{
// 					Id:          "e1",
// 					Company:     "Tech Corp",
// 					Position:    "Senior Developer",
// 					StartDate:   "2020-01",
// 					EndDate:     "Present",
// 					Description: "Lead development of cloud-based applications using React and Node.js",
// 				},
// 				{
// 					Id:          "e2",
// 					Company:     "Digital Solutions",
// 					Position:    "Full Stack Developer",
// 					StartDate:   "2016-06",
// 					EndDate:     "2019-12",
// 					Description: "Built and maintained enterprise web applications",
// 				},
// 			},
// 			Education: []models.Education{
// 				{
// 					Id:             "ed1",
// 					Institution:    "University of Technology",
// 					Degree:         "Bachelor of Science",
// 					Field:          "Computer Science",
// 					GraduationDate: "2016",
// 				},
// 			},
// 			CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
// 			UpdatedAt: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
// 		}
// 	} else if id == "2" {
// 		profile = models.Profile{
// 			Id:      "2",
// 			Name:    "Michael Chen",
// 			Email:   "michael.chen@email.com",
// 			Phone:   "+1 (555) 987-6543",
// 			Title:   "UX/UI Designer",
// 			Summary: "Creative designer with 5+ years of experience in user interface and user experience design for web and mobile applications.",
// 			Skills:  []string{"Figma", "Adobe XD", "Sketch", "HTML/CSS", "JavaScript", "Prototyping", "User Research", "Wireframing"},
// 			Experience: []models.Experience{
// 				{
// 					Id:          "e3",
// 					Company:     "Design Studio",
// 					Position:    "Senior UX Designer",
// 					StartDate:   "2022-03",
// 					EndDate:     "Present",
// 					Description: "Lead UX design for multiple client projects",
// 				},
// 			},
// 			Education: []models.Education{
// 				{
// 					Id:             "ed2",
// 					Institution:    "Art & Design College",
// 					Degree:         "Bachelor of Arts",
// 					Field:          "Graphic Design",
// 					GraduationDate: "2019",
// 				},
// 			},
// 			CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
// 			UpdatedAt: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
// 		}
// 	} else {
// 		c.JSON(404, gin.H{"error": "Profile not found"})
// 		return
// 	}

// 	c.JSON(200, profile)
// }

func (s *HttpSvc) HandleListProfiles(c *gin.Context) {
	user, _ := authorization.User(c)
	profiles, err := s.Service.ListProfiles(user)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, profiles)
}

func (s *HttpSvc) HandleCreateProfile(c *gin.Context) {
	user, _ := authorization.User(c)
	var profile models.Profile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	_, err := s.Service.CreateProfile(user, &profile)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	// Call matchingSvc to embed the experience data
	matchingSvc := clientSide.NewMatchingSvc()
	_, err = matchingSvc.GenerateWorkExpEmbeddings(profile.Id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{
		"message": "CV created successfully"})
}

func (s *HttpSvc) HandleUpdateProfile(c *gin.Context) {
	user, _ := authorization.User(c)
	var profile models.Profile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	profile.Id = c.Param("id")

	err := s.Service.UpdateProfile(user, profile)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"message": "CV updated successfully"})
}

func (s *HttpSvc) HandleDeleteProfile(c *gin.Context) {
	user, _ := authorization.User(c)
	id := c.Param("id")
	err := s.Service.DeleteProfile(user, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"message": "CV deleted successfully"})
}

func writeServiceError(c *gin.Context, err error) {
	if errors.Is(err, authorization.ErrForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
