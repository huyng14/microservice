package httpServerSvc

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type HttpSvc struct {
}

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

func (s *HttpSvc) HandleGetPerson(c *gin.Context) {
	id := c.Param("id")
	var profile Profile

	if id == "1" {
		profile = Profile{
			Id:      "1",
			Name:    "Sarah Johnson",
			Email:   "sarah.johnson@email.com",
			Phone:   "+1 (555) 123-4567",
			Title:   "Senior Full Stack Developer",
			Summary: "Experienced software engineer with 8+ years in web development, specializing in React, Node.js, and cloud technologies.",
			Skills:  []string{"React", "Node.js", "TypeScript", "AWS", "PostgreSQL", "Docker", "GraphQL", "REST APIs"},
			Experience: []Experience{
				{
					Id:          "e1",
					Company:     "Tech Corp",
					Position:    "Senior Developer",
					StartDate:   "2020-01",
					EndDate:     "Present",
					Description: "Lead development of cloud-based applications using React and Node.js",
				},
				{
					Id:          "e2",
					Company:     "Digital Solutions",
					Position:    "Full Stack Developer",
					StartDate:   "2016-06",
					EndDate:     "2019-12",
					Description: "Built and maintained enterprise web applications",
				},
			},
			Education: []Education{
				{
					Id:             "ed1",
					Institution:    "University of Technology",
					Degree:         "Bachelor of Science",
					Field:          "Computer Science",
					GraduationDate: "2016",
				},
			},
			CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		}
	} else if id == "2" {
		profile = Profile{
			Id:      "2",
			Name:    "Michael Chen",
			Email:   "michael.chen@email.com",
			Phone:   "+1 (555) 987-6543",
			Title:   "UX/UI Designer",
			Summary: "Creative designer with 5+ years of experience in user interface and user experience design for web and mobile applications.",
			Skills:  []string{"Figma", "Adobe XD", "Sketch", "HTML/CSS", "JavaScript", "Prototyping", "User Research", "Wireframing"},
			Experience: []Experience{
				{
					Id:          "e3",
					Company:     "Design Studio",
					Position:    "Senior UX Designer",
					StartDate:   "2022-03",
					EndDate:     "Present",
					Description: "Lead UX design for multiple client projects",
				},
			},
			Education: []Education{
				{
					Id:             "ed2",
					Institution:    "Art & Design College",
					Degree:         "Bachelor of Arts",
					Field:          "Graphic Design",
					GraduationDate: "2019",
				},
			},
			CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		}
	} else {
		c.JSON(404, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(200, profile)
}

func (s *HttpSvc) HandleListProfiles(c *gin.Context) {
	profiles := []Profile{}
	profiles = append(profiles, Profile{
		Id:      "1",
		Name:    "Sarah Johnson",
		Email:   "sarah.johnson@email.com",
		Phone:   "+1 (555) 123-4567",
		Title:   "Senior Full Stack Developer",
		Summary: "Experienced software engineer with 8+ years in web development, specializing in React, Node.js, and cloud technologies.",
		Skills:  []string{"React", "Node.js", "TypeScript", "AWS", "PostgreSQL", "Docker", "GraphQL", "REST APIs"},
		Experience: []Experience{
			{
				Id:          "e1",
				Company:     "Tech Corp",
				Position:    "Senior Developer",
				StartDate:   "2020-01",
				EndDate:     "Present",
				Description: "Lead development of cloud-based applications using React and Node.js",
			},
			{
				Id:          "e2",
				Company:     "Digital Solutions",
				Position:    "Full Stack Developer",
				StartDate:   "2016-06",
				EndDate:     "2019-12",
				Description: "Built and maintained enterprise web applications",
			},
		},
		Education: []Education{
			{
				Id:             "ed1",
				Institution:    "University of Technology",
				Degree:         "Bachelor of Science",
				Field:          "Computer Science",
				GraduationDate: "2016",
			},
		},
		CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	})
	profiles = append(profiles, Profile{
		Id:      "2",
		Name:    "Michael Chen",
		Email:   "michael.chen@email.com",
		Phone:   "+1 (555) 987-6543",
		Title:   "UX/UI Designer",
		Summary: "Creative designer with 5+ years of experience in user interface and user experience design for web and mobile applications.",
		Skills:  []string{"Figma", "Adobe XD", "Sketch", "HTML/CSS", "JavaScript", "Prototyping", "User Research", "Wireframing"},
		Experience: []Experience{
			{
				Id:          "e3",
				Company:     "Design Studio",
				Position:    "Senior UX Designer",
				StartDate:   "2022-03",
				EndDate:     "Present",
				Description: "Lead UX design for multiple client projects",
			},
		},
		Education: []Education{
			{
				Id:             "ed2",
				Institution:    "Art & Design College",
				Degree:         "Bachelor of Arts",
				Field:          "Graphic Design",
				GraduationDate: "2019",
			},
		},
		CreatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
	})
	log.Println("length of profiles = ", len(profiles))
	c.JSON(200, profiles)
}

func (s *HttpSvc) HandleCreateProfile(c *gin.Context) {

}

func (s *HttpSvc) HandleUpdateProfile(c *gin.Context) {

}
