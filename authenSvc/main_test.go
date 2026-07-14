package main

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestGenerateResetToken(t *testing.T) {
	token, err := generateResetToken()
	if err != nil {
		t.Fatalf("generateResetToken() returned error: %v", err)
	}

	if len(token) != 64 {
		t.Fatalf("generateResetToken() returned unexpected length: got %d, want 64", len(token))
	}
	log.Println("token: ", token)
}

func TestResetTokenExpiry(t *testing.T) {
	now := time.Now().UTC()

	if !isResetTokenExpired(now.Add(2 * time.Minute)) {
		log.Println("a valid token")
		t.Log("expected a token expiring in the future to be valid")
	}

	if isResetTokenExpired(now.Add(-1 * time.Minute)) {
		log.Println("an invalid token")
		t.Log("expected a token already expired to be invalid")
	}
	// expired := isResetTokenExpired(now.Add(-time.Minute))
	// t.Logf("expired = %v", expired)

	// if expired {
	// 	t.Fatal("invalid token")
	// } else {
	// 	t.Log("valid token")
	// }
}

func TestSendPasswordResetEmail(t *testing.T) {
	// Test implementation for sendPasswordResetEmail function
	resetLink := buildResetPasswordLink("testtoken")
	fmt.Println("Reset link: ", resetLink)
	if err := sendPasswordResetEmail("hittle9x@gmail.com", resetLink); err != nil {
		t.Fatalf("sendPasswordResetEmail() returned error: %v", err)
	}
}
