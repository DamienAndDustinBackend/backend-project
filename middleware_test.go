package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
