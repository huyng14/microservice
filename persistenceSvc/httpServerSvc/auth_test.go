package httpServerSvc

import (
	"context"
	"log"
	mongodb "microservice/mongoDB"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type fakeUserRoleStore struct {
	userID   string
	role     string
	mongoSvc *mongodb.MongoSvc
}

func (f *fakeUserRoleStore) GetUserRole(_ context.Context, userID string) (string, error) {
	f.userID = userID
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://skylab:skylab@consultatantaimatch.ftecqos.mongodb.net/"))
	if err != nil {
		return "", err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return "", err
	}

	log.Println("Connected to MongoDB!")
	mongoSvc := &mongodb.MongoSvc{Client: client}
	f.mongoSvc = mongoSvc

	role, err := mongoSvc.GetUserRole(ctx, userID)
	if err != nil {
		return "", err
	}
	f.role = role
	log.Println("(f *fakeUserRoleStore) GetUserRole() role = ", role)

	return f.role, nil
}

func TestAuthMiddlewareLoadsRoleFromStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "dev-secret-change-me")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "6a61cacfa801a87dca9780e3",
		"exp": time.Now().Add(time.Hour).Unix(),
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
