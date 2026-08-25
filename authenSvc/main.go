package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email               string             `bson:"email" json:"email"`
	Password            string             `bson:"password" json:"-"`
	ResetToken          string             `bson:"resetToken,omitempty" json:"-"`
	ResetTokenExpiresAt time.Time          `bson:"resetTokenExpiresAt,omitempty" json:"-"`
	CreatedAt           time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type signinRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password" binding:"required,min=6"`
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
	jwtSecret       = []byte("dev-secret-change-me")
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found")
	}
	jwtSecretEnv := os.Getenv("JWT_SECRET")
	if jwtSecretEnv == "" {
		log.Println("Warning: JWT_SECRET is not set. Using default secret. This is not secure for production.")
	} else {
		jwtSecret = []byte(jwtSecretEnv)
	}
	// log.Println("jwtSecret", string(jwtSecret))

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
		cfg.AllowOrigins = []string{"http://localhost:5173"}
	} else {
		parts := strings.Split(originsEnv, ";")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		cfg.AllowOrigins = parts
	}
	log.Println("CORS: ", cfg.AllowOrigins)
	router.Use(cors.New(cfg))

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "Ok for health check"})
	})

	router.POST("/signup", signupHandler())
	router.POST("/login", loginHandler())
	router.POST("/forgot-password", forgotPasswordHandler())
	router.POST("/new-password", resetPasswordHandler())

	protected := router.Group("/")
	protected.Use(authMiddleware())
	protected.GET("/profile", profileHandler())

	port := getEnv("PORT", "9001")
	log.Printf("auth service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func initMongo() error {
	uri := getEnv("MONGODB_URI", "")
	dbName := getEnv("MONGODB_DB", "project")
	collectionName := getEnv("MONGODB_COLLECTION", "usersdb")

	if uri == "" && dbName == "" && collectionName == "" {
		log.Println("MongoDB connection details not provided. Skipping MongoDB initialization.")
		return nil
	}

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

		// If the email is not from @adabeat.com, return an error
		if !strings.HasSuffix(strings.ToLower(req.Email), "@adabeat.com") {
			c.JSON(http.StatusForbidden, gin.H{"error": "only @adabeat.com emails are allowed"})
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
				"error": "credentials: " + err.Error()})
			return
		}

		if usersCollection == nil {
			// c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Invalid credentials.",
				"error": "database not available."})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user User
		err := usersCollection.FindOne(ctx, bson.M{"email": strings.ToLower(req.Email)}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials.",
				"error": "Invalid email or password entered."})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials.",
				"error": "Invalid email or password entered."})
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

func forgotPasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req forgotPasswordRequest
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

		email := strings.ToLower(req.Email)
		var user User
		err := usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "The email address is not registered with us. Please check and try again."})
			return
		}

		token, err := generateResetToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create reset token"})
			return
		}

		expiresAt := time.Now().UTC().Add(15 * time.Minute)
		_, err = usersCollection.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
			"$set": bson.M{
				"resetToken":          token,
				"resetTokenExpiresAt": expiresAt,
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save reset token"})
			return
		}

		resetLink := buildResetPasswordLink(token)
		if err := sendPasswordResetEmail(email, resetLink); err != nil {
			log.Printf("could not send password reset email: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "A reset link has been sent to your email address. Please check your inbox."})
	}
}

func resetPasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req resetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token, exists := c.GetQuery("token")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
			return
		}
		req.Token = token

		if usersCollection == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user User
		err := usersCollection.FindOne(ctx, bson.M{"resetToken": req.Token}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired reset token"})
			return
		}

		if user.ResetToken == "" || user.ResetToken != req.Token || isResetTokenExpired(user.ResetTokenExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired reset token"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
			return
		}

		_, err = usersCollection.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
			"$set": bson.M{
				"password":  string(hashedPassword),
				"updatedAt": time.Now().UTC(),
			},
			"$unset": bson.M{
				"resetToken":          "",
				"resetTokenExpiresAt": "",
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
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

func generateResetToken() (string, error) {

	aes256Key := make([]byte, 32) // 256 bits
	if _, err := rand.Read(aes256Key); err != nil {
		log.Println("generateResetToken() error: ", err)
	}

	// aes256Hex := hex.EncodeToString(aes256Key)
	// log.Println("AES-256 Key (Hex):", aes256Hex)
	return hex.EncodeToString(aes256Key), nil
}

func isResetTokenExpired(expiresAt time.Time) bool {
	return !expiresAt.IsZero() && time.Now().UTC().After(expiresAt)
}

func buildResetPasswordLink(token string) string {
	baseURL := getEnv("FRONTEND_URL", "http://localhost:3000/reset-password")
	if strings.Contains(baseURL, "?") {
		return fmt.Sprintf("%s&token=%s", baseURL, token)
	}
	return fmt.Sprintf("%s?token=%s", baseURL, token)
}

func sendPasswordResetEmail(toEmail, resetLink string) error {
	host := getEnv("SMTP_HOST", "smtp.gmail.com")
	if host == "" {
		return fmt.Errorf("smtp host not configured")
	}

	smtpPort := getEnv("SMTP_PORT", "587")
	username := getEnv("SMTP_USERNAME", "")
	password := getEnv("SMTP_PASSWORD", "")
	from := getEnv("SMTP_FROM", username)
	if username == "" || password == "" {
		return fmt.Errorf("smtp credentials not configured")
	}

	auth := smtp.PlainAuth("", username, password, host)
	message := fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: [TalentMatch] Reset your password\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n"+

			`<!DOCTYPE html>
		<html>
		<head>
		<meta charset="UTF-8">
		<title>Reset Password</title>
		</head>
		<body style="font-family: Arial, Helvetica, sans-serif; background-color:#f4f4f4; margin:0; padding:40px;">
			<div style="max-width:600px; margin:auto; background:#ffffff; padding:40px; border-radius:8px;">

				<h2 style="color:#333333;">Reset Your Password</h2>

				<p>Hello,</p>

				<p>We received a request to reset the password for your <strong>TalentMatch</strong> account.</p>

				<p>Click the button below to create a new password:</p>

				<p style="text-align:center; margin:35px 0;">
					<a href="%s"
						style="
							background-color:#2563eb;
							color:#ffffff;
							padding:14px 28px;
							text-decoration:none;
							border-radius:6px;
							font-weight:bold;
							display:inline-block;">
						Reset Password
					</a>
				</p>
				<p>This link will expire after 15 minutes.</p>
				<p>If the button doesn't work, copy and paste the following link into your browser:</p>

				<p>
					<a href="%s">%s</a>
				</p>

				<hr style="border:none; border-top:1px solid #e5e5e5; margin:30px 0;">

				<p style="color:#666666; font-size:14px;">
					If you did not request a password reset, you can safely ignore this email.
					Your password will remain unchanged.
				</p>

				<p style="color:#666666; font-size:14px;">
					Thanks,<br>
					The TalentMatch Team
				</p>

			</div>
		</body>
		</html>`,
		toEmail,
		resetLink,
		resetLink,
		resetLink,
	)
	addr := net.JoinHostPort(host, smtpPort)
	return smtp.SendMail(addr, auth, from, []string{toEmail}, []byte(message))
}

func generateToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"aud": "api",
		"iat": now.Unix(),
		"exp": now.Add(15 * time.Minute).Unix(),
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
