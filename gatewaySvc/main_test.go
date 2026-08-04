package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func externalToken(t *testing.T, secret, subject string) string {
	t.Helper()
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: subject, Audience: jwt.ClaimStrings{externalAudience},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestGatewayReplacesExternalTokenWithInternalIdentity(t *testing.T) {
	const external, internal = "external-secret", "internal-secret"
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		raw, err := bearerToken(r.Header.Get("Authorization"))
		if err != nil {
			t.Error(err)
			return response(http.StatusUnauthorized), nil
		}
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
			return []byte(internal), nil
		}, jwt.WithIssuer(internalIssuer), jwt.WithAudience(internalAudience))
		if err != nil || !token.Valid || claims.Subject != "user-123" {
			t.Errorf("invalid internal token: claims=%+v err=%v", claims, err)
			return response(http.StatusUnauthorized), nil
		}
		return response(http.StatusNoContent), nil
	})
	target, _ := url.Parse("http://persistence")
	handler := gatewayHandler(config{
		externalSecret: []byte(external), internalSecret: []byte(internal),
		persistenceURL: target, rateLimit: 2, rateWindow: time.Minute, transport: transport,
	})

	request := httptest.NewRequest(http.MethodPut, "/profile/cv-1", nil)
	request.Header.Set("Authorization", "Bearer "+externalToken(t, external, "user-123"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestGatewayRateLimitsBySubject(t *testing.T) {
	target, _ := url.Parse("http://example.invalid")
	handler := gatewayHandler(config{
		externalSecret: []byte("external"), internalSecret: []byte("internal"),
		persistenceURL: target, rateLimit: 1, rateWindow: time.Minute,
		transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusNoContent), nil
		}),
	})
	token := externalToken(t, "external", "user-123")
	for i, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodGet, "/listprofiles", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("request %d status = %d, want %d", i+1, response.Code, want)
		}
	}
}

func TestGatewayDoesNotProxyUnknownRoutes(t *testing.T) {
	target, _ := url.Parse("http://persistence")
	proxied := false
	handler := gatewayHandler(config{
		externalSecret: []byte("external"), internalSecret: []byte("internal"),
		persistenceURL: target, rateLimit: 10, rateWindow: time.Minute,
		transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			proxied = true
			return response(http.StatusNoContent), nil
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/not-a-persistence-route", nil)
	request.Header.Set("Authorization", "Bearer "+externalToken(t, "external", "user-123"))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)

	if result.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusNotFound)
	}
	log.Println("handler.ServeHTTP returned status", result.Code)
	if proxied {
		t.Fatal("unknown route was sent to persistence")
	}
}

func TestGatewayAddsCorsHeadersForPreflightRequests(t *testing.T) {
	target, _ := url.Parse("http://persistence")
	handler := gatewayHandler(config{
		externalSecret: []byte("external"), internalSecret: []byte("internal"),
		persistenceURL: target, rateLimit: 10, rateWindow: time.Minute,
		transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusNoContent), nil
		}),
	})

	request := httptest.NewRequest(http.MethodOptions, "/profile", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)

	if result.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusNoContent)
	}
	if got := result.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("access-control-allow-origin = %q, want %q", got, "http://localhost:5173")
	}
}

func TestGatewayDropsUpstreamCorsHeaders(t *testing.T) {
	target, _ := url.Parse("http://persistence")
	handler := gatewayHandler(config{
		externalSecret: []byte("external"), internalSecret: []byte("internal"),
		persistenceURL: target, rateLimit: 10, rateWindow: time.Minute,
		transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			resp := response(http.StatusNoContent)
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			resp.Header.Set("Access-Control-Allow-Credentials", "true")
			return resp, nil
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/listjobs", nil)
	request.Header.Set("Authorization", "Bearer "+externalToken(t, "external", "user-123"))
	request.Header.Set("Origin", "http://localhost:5173")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)

	if got := result.Header().Values("Access-Control-Allow-Origin"); len(got) != 1 || got[0] != "http://localhost:5173" {
		t.Fatalf("access-control-allow-origin values = %v, want [http://localhost:5173]", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
}
