package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


func TestGetRole(t *testing.T) {
	// Test admin role
	if role := getRole("damien.z.hall@gmail.com"); role != "admin" {
		t.Errorf("getRole() = %v, want %v", role, "admin")
	}

	// Test default role
	if role := getRole("test@example.com"); role != "default" {
		t.Errorf("getRole() = %v, want %v", role, "default")
	}
}

func TestGenerateAndVerifyJWT(t *testing.T) {
	email := "test@example.com"
	tokenString, err := GenerateJWT(email)
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v, wantErr %v", err, false)
	}

	if len(tokenString) == 0 {
		t.Fatalf("GenerateJWT() token is empty")
	}

	token, err := VerifyJWT(tokenString)
	if err != nil {
		t.Fatalf("VerifyJWT() error = %v, wantErr %v", err, false)
	}

	if !token.Valid {
		t.Fatalf("VerifyJWT() token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("VerifyJWT() claims are not of type jwt.MapClaims")
	}

	if claims["sub"] != email {
		t.Errorf("VerifyJWT() subject = %v, want %v", claims["sub"], email)
	}

	if claims["aud"] != "default" {
		t.Errorf("VerifyJWT() audience = %v, want %v", claims["aud"], "default")
	}
}

func TestVerifyJWT_InvalidToken(t *testing.T) {
	_, err := VerifyJWT("invalid-token-string")
	if err == nil {
		t.Errorf("VerifyJWT() error = nil, wantErr %v", true)
	}
}

func TestVerifyJWT_ExpiredToken(t *testing.T) {
	// Create a token that expired 1 hour ago
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test@example.com",
		"iss": "snippet-app",
		"aud": "default",
		"exp": time.Now().Add(-time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})

	secret := os.Getenv("JWT_SECRET")
	tokenString, err := claims.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign expired token: %v", err)
	}

	_, err = VerifyJWT(tokenString)
	if err == nil {
		t.Errorf("VerifyJWT() with expired token should have failed, but it did not")
	}
}
