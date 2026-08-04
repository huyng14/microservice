package httpServerSvc

import (
	"context"
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

// AuthMiddleware trusts only short-lived identity tokens minted by the gateway.
// Roles are deliberately loaded by persistence and never accepted from claims.
func AuthMiddleware(users UserRoleStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtSecret := []byte("internal-secret-change-me")
		if len(jwtSecret) == 0 {
			log.Print("INTERNAL_IDENTITY_SECRET is not configured")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable"})
			return
		}

		authorizationHeader := c.GetHeader("Authorization")
		parts := strings.Split(authorizationHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, authorization.ErrForbidden
			}
			return jwtSecret, nil
		}, jwt.WithIssuer("gateway"), jwt.WithAudience("persistence"),
			jwt.WithExpirationRequired(), jwt.WithIssuedAt())
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token by parse"})
			return
		}

		userID := claims.Subject
		if userID == "" {
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
