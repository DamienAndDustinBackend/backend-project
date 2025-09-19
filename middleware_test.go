package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	db := setupDatabase()
	app := App{db: db}
	// Create a user
	user := User{
		Email:    "test@example.com",
		Password: "secret",
	}
	err := gorm.G[User](db).Create(t.Context(), &user)
	if err != nil {
		panic(err)
	}

	// Create a valid token
	token, err := GenerateJWT("test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate token for test: %v", err)
	}

	// Add the token to the request cookies
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "token",
		Value: token,
	})

	// Create a dummy handler to check if the middleware calls c.Next()
	nextCalled := false
	r.GET("/", app.AuthMiddleware, func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	// Execute the request
	r.ServeHTTP(w, c.Request)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("AuthMiddleware with valid token returned status %d, want %d", w.Code, http.StatusOK)
	}

	if !nextCalled {
		t.Errorf("AuthMiddleware with valid token did not call the next handler")
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	// Create a dummy handler
	db := setupDatabase()
	app := App{db: db}
	r.GET("/", app.AuthMiddleware, func(c *gin.Context) {
		// This should not be called
		t.Error("Next handler was called, but should have been aborted")
	})

	// Execute the request
	r.ServeHTTP(w, c.Request)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("AuthMiddleware with no token returned status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	// Add an invalid token to the request cookies
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "token",
		Value: "this-is-not-a-valid-jwt",
	})

	// Create a dummy handler
	db := setupDatabase()
	app := App{db: db}
	r.GET("/", app.AuthMiddleware, func(c *gin.Context) {
		// This should not be called
		t.Error("Next handler was called, but should have been aborted")
	})

	// Execute the request
	r.ServeHTTP(w, c.Request)

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("AuthMiddleware with invalid token returned status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

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

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Fatalf("VerifyJWT() claims are not of type jwt.RegisteredClaims")
	}

	if claims.Subject != email {
		t.Errorf("VerifyJWT() subject = %v, want %v", claims.Subject, email)
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
