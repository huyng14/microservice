package httpServerSvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type fakeUserRoleStore struct {
	userID string
	role   string
}

func (f *fakeUserRoleStore) GetUserRole(_ context.Context, userID string) (string, error) {
	f.userID = userID
	f.role = "CONSULTANT"
	return f.role, nil
}

func TestAuthMiddlewareLoadsRoleFromStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("INTERNAL_IDENTITY_SECRET", "dev-secret-change-me")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "6a61cacfa801a87dca9780e3",
		Issuer:    "gateway",
		Audience:  jwt.ClaimStrings{"persistence"},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	})
	tokenValue, err := token.SignedString([]byte("dev-secret-change-me"))
	if err != nil {
		t.Fatal(err)
	}

	store := &fakeUserRoleStore{}
	router := gin.New()
	router.Use(AuthMiddleware(store))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+tokenValue)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	t.Logf("role lookup userID = %q", store.userID)
	t.Logf("role lookup role = %q", store.role)
}
