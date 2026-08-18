package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	externalAudience = "api"
	internalIssuer   = "gateway"
	internalAudience = "persistence"
)

type config struct {
	externalSecret  []byte
	internalSecret  []byte
	persistenceURL  *url.URL
	matchingURL     *url.URL
	authenURL       *url.URL
	rateLimit       int
	rateWindow      time.Duration
	transport       http.RoundTripper
	corsOrigins     []string
	allowAllOrigins bool
}

type window struct {
	start time.Time
	count int
}

type limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]window
}

func newLimiter(limit int, duration time.Duration) *limiter {
	return &limiter{limit: limit, window: duration, clients: make(map[string]window)}
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	current := l.clients[key]
	if current.start.IsZero() || now.Sub(current.start) >= l.window {
		l.clients[key] = window{start: now, count: 1}
		return true
	}
	if current.count >= l.limit {
		return false
	}
	current.count++
	l.clients[key] = current
	return true
}

const subjectContextKey = "subject"

func gatewayHandler(cfg config) *gin.Engine {

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(corsMiddleware(cfg))

	// Public auth routes (no authentication required)
	authentication := createProxyHandler(cfg.authenURL, cfg.transport, "authentication")
	router.POST("/signup", authenticationMiddleware(cfg.externalSecret), authGetFirstToken(), signupHandler(authentication))
	router.POST("/login", authenticationMiddleware(cfg.externalSecret), authGetFirstToken(), loginHandler(authentication))
	router.POST("/forgot-password", forgotPasswordHandler(authentication))
	router.POST("/new-password", resetPasswordHandler(authentication))

	// Protected routes with authentication
	router.Use(authenticationMiddleware(cfg.externalSecret))
	router.Use(rateLimitMiddleware(newLimiter(cfg.rateLimit, cfg.rateWindow)))
	router.Use(internalIdentityMiddleware(cfg.internalSecret))

	persistence := createProxyHandler(cfg.persistenceURL, cfg.transport, "persistence")
	matching := createProxyHandler(cfg.matchingURL, cfg.transport, "matching")

	// Gateway :8080                    Persistence :9000
	router.GET("/listprofiles", persistence)   // GET    /listprofiles -> GET    /listprofiles
	router.POST("/profile", persistence)       // POST   /profile      -> POST   /profile
	router.PUT("/profile/:id", persistence)    // PUT    /profile/:id  -> PUT    /profile/:id
	router.DELETE("/profile/:id", persistence) // DELETE /profile/:id  -> DELETE /profile/:id
	router.GET("/listjobs", persistence)       // GET    /listjobs     -> GET    /listjobs
	router.POST("/job", persistence)           // POST   /job          -> POST   /job
	router.PUT("/job/:id", persistence)        // PUT    /job/:id      -> PUT    /job/:id
	router.DELETE("/job/:id", persistence)     // DELETE /job/:id      -> DELETE /job/:id

	// Gateway :8080                    Matching :9030
	router.POST("/matching/:consultant_id", matching)
	router.POST("/embeddingmodel/consultant/:consultant_id", matching)
	router.POST("/embeddingmodel/job/:job_id", matching)

	return router
}

func corsMiddleware(cfg config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowedOrigins := cfg.corsOrigins
		if len(allowedOrigins) == 0 {
			allowedOrigins = []string{"http://localhost:5173"}
		}

		allowed := cfg.allowAllOrigins
		if !allowed {
			for _, candidate := range allowedOrigins {
				if candidate == origin {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			c.Next()
			return
		}

		if cfg.allowAllOrigins {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func proxyHandler(proxy *httputil.ReverseProxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func createProxyHandler(target *url.URL, transport http.RoundTripper, serviceName string) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)
	if transport != nil {
		proxy.Transport = transport
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("%s request failed: %v", serviceName, err)
		writeJSON(w, http.StatusBadGateway, serviceName+" unavailable")
	}
	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authGetFirstToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.Set(subjectContextKey, "")
		}
		c.Next()
	}
}

func signupHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return handler
}

func loginHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return handler
}

func forgotPasswordHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c)
	}
}

func resetPasswordHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c)
	}
}

func authenticationMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject, err := validateExternalJWT(c.GetHeader("Authorization"), secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			return
		}
		c.Set(subjectContextKey, subject)
		c.Next()
	}
}

func rateLimitMiddleware(limit *limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject := c.GetString(subjectContextKey)
		if !limit.allow(subject, time.Now()) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func internalIdentityMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject := c.GetString(subjectContextKey)
		identity, err := signInternalIdentity(subject, secret, time.Now())
		if err != nil {
			log.Printf("could not sign internal identity: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Request.Header.Set("Authorization", "Bearer "+identity)
		c.Request.Header.Set("X-Forwarded-User", subject)
		c.Next()
	}
}

func validateExternalJWT(header string, secret []byte) (string, error) {
	raw, err := bearerToken(header)
	if err != nil {
		return "", err
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithAudience(externalAudience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.Subject == "" {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}

func signInternalIdentity(subject string, secret []byte, now time.Time) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    internalIssuer,
		Audience:  jwt.ClaimStrings{internalAudience},
		ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
		ID:        hex.EncodeToString(nonce),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("missing bearer token")
	}
	return parts[1], nil
}

func writeJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func env(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", errors.New(name + " is required")
	}
	return value, nil
}

func getEnv(key, fallback string) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}
	return fallback, nil
}

func loadConfig() (config, error) {
	external, err := getEnv("EXTERNAL_IDENTITY_SECRET", "dev-secret-change-me")
	if err != nil {
		return config{}, err
	}
	internal, err := getEnv("INTERNAL_IDENTITY_SECRET", "internal-secret-change-me")
	if err != nil {
		return config{}, err
	}
	persistence, err := getEnv("PERSISTENCE_HTTP_URL", "http://localhost:9000")
	if err != nil {
		return config{}, err
	}
	persistenceURL, err := url.Parse(persistence)
	if err != nil {
		return config{}, err
	}
	matching, err := getEnv("MATCHING_HTTP_URL", "http://localhost:9020")
	if err != nil {
		return config{}, err
	}
	matchingURL, err := url.Parse(matching)
	if err != nil {
		return config{}, err
	}

	authen, err := getEnv("AUTHEN_HTTP_URL", "http://localhost:9001")
	if err != nil {
		return config{}, err
	}
	authenURL, err := url.Parse(authen)
	if err != nil {
		return config{}, err
	}

	originsEnv := os.Getenv("CORS_ALLOW_ORIGINS")
	var corsOrigins []string
	allowAllOrigins := false
	if originsEnv == "*" {
		allowAllOrigins = true
	} else if originsEnv == "" {
		corsOrigins = []string{"http://localhost:5173"}
	} else {
		parts := strings.Split(originsEnv, ";")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		corsOrigins = parts
	}

	return config{
		externalSecret: []byte(external), internalSecret: []byte(internal),
		persistenceURL: persistenceURL, matchingURL: matchingURL, authenURL: authenURL,
		rateLimit: 60, rateWindow: time.Minute,
		corsOrigins: corsOrigins, allowAllOrigins: allowAllOrigins,
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              ":8080",
		Handler:           gatewayHandler(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("gateway listening on :8080")
	log.Fatal(server.ListenAndServe())
}
