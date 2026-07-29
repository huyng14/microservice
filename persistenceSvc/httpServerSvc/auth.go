package httpServerSvc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"microservice/authorization"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type UserRoleStore interface {
	GetUserRole(ctx context.Context, userID string) (string, error)
}

// AuthMiddleware validates a Bearer JWT from the Authorization header.
// It expects a HMAC-signed token and uses the JWT_SECRET environment variable.
func AuthMiddleware(users UserRoleStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtSecret := []byte("dev-secret-change-me")

		authorizationHeader := c.GetHeader("Authorization")
		parts := strings.Split(authorizationHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
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

		roleValue, err := users.GetUserRole(c.Request.Context(), userID)
		if err != nil {
			log.Printf("could not load role for user %s: %v", userID, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if roleValue == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user has no role"})
			return
		}
		role := authorization.Role(strings.ToUpper(roleValue))
		if _, known := authorization.RolePermissions[role]; !known {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user has invalid role"})
			return
		}

		c.Set("user", authorization.UserContext{UserID: userID, Role: role})
		c.Next()
	}
}
