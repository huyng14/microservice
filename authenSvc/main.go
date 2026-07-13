package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type signinRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

var (
	usersCollection *mongo.Collection
	jwtSecret       = []byte(getEnv("JWT_SECRET", "dev-secret-change-me"))
)

func main() {
	if err := initMongo(); err != nil {
		log.Printf("mongo connection warning: %v", err)
	}

	router := gin.Default()

	// Enable CORS so Vue (port 5173) can call Go (port 9000)
	cfg := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}

	// Read allowed origins from env var CORS_ALLOW_ORIGINS (comma-separated).
	// Use "*" to allow all origins (sets AllowAllOrigins=true).
	originsEnv := os.Getenv("CORS_ALLOW_ORIGINS")
	if originsEnv == "*" {
		cfg.AllowAllOrigins = true
	} else if originsEnv == "" {
		// fallback default used previously
		cfg.AllowOrigins = []string{"http://localhost:3000"}
	} else {
		parts := strings.Split(originsEnv, ";")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		cfg.AllowOrigins = parts
	}

	router.Use(cors.New(cfg))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/signup", signupHandler())
	router.POST("/login", loginHandler())

	protected := router.Group("/")
	protected.Use(authMiddleware())
	protected.GET("/profile", profileHandler())

	port := getEnv("PORT", "8282")
	log.Printf("auth service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func initMongo() error {
	uri := getEnv("MONGODB_URI", "mongodb+srv://skylab:skylab@consultatantaimatch.ftecqos.mongodb.net/")
	dbName := getEnv("MONGODB_DB", "project")
	collectionName := getEnv("MONGODB_COLLECTION", "usersdb")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	usersCollection = client.Database(dbName).Collection(collectionName)

	_, err = usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"email": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	return nil
}

func signupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if usersCollection == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing User
		err := usersCollection.FindOne(ctx, bson.M{"email": strings.ToLower(req.Email)}).Decode(&existing)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		if err != nil && err != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not check existing user"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
			return
		}

		user := User{
			Email:     strings.ToLower(req.Email),
			Password:  string(hashedPassword),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		result, err := usersCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
			return
		}

		user.ID = result.InsertedID.(primitive.ObjectID)
		token, err := generateToken(user.ID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
			return
		}

		c.JSON(http.StatusCreated, authResponse{
			Token: token,
			User:  mapUser(user),
		})
	}
}

func loginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signinRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(),
				"errors": gin.H{"credentials": err.Error()}})
			return
		}

		if usersCollection == nil {
			// c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Invalid credentials.",
				"errors": gin.H{"credentials": "database not available."}})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user User
		err := usersCollection.FindOne(ctx, bson.M{"email": strings.ToLower(req.Email)}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials.",
				"errors": gin.H{"credentials": "Invalid email or password entered."}})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials.",
				"errors": gin.H{"credentials": "Invalid email or password entered."}})
			return
		}

		token, err := generateToken(user.ID.Hex())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
			return
		}

		c.JSON(http.StatusOK, authResponse{
			Token: token,
			User:  mapUser(user),
		})
	}
}

func profileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userTmp User
		if err := c.ShouldBindJSON(&userTmp); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if userTmp.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email not provided"})
			return
		}

		if usersCollection == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user User
		err := usersCollection.FindOne(ctx, bson.M{"email": userTmp.Email}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": mapUser(user)})
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		parts := strings.Split(authorization, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token by parse"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token by claims"})
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func mapUser(user User) userResponse {
	return userResponse{
		ID:        user.ID.Hex(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
