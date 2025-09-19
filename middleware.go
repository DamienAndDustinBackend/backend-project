package main

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func getRole(email string) string {
	admins := []string{"damien.z.hall@gmail.com", "dustin.alandzes@hp.com"}
	if slices.Contains(admins, email) {
		return "admin"
	}

	return "default"
}

func GenerateJWT(email string) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   email,
		Issuer:    "backend-project",
		Audience:  []string{}, // optional
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})

	tokenString, err := claims.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {
	// https://pkg.go.dev/github.com/golang-jwt/jwt/v5#Parse
	// https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
	// It's important to validate the alg is what you expect
	// TODO: test with alg: none w/o ParseWithClaims
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

const ContextUserKey = "user"

func (app *App) AuthMiddleware(c *gin.Context) {
	// find the jwt from cookies
	tokenString, err := c.Cookie("token")

	if err != nil {
		fmt.Println("JWT missing in cookies")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// verify jwt
	token, err := VerifyJWT(tokenString)

	if err != nil {
		fmt.Printf("JWT verification failed: %v\n", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	fmt.Printf("JWT verified. Claims: %+v\n", token.Claims)

	user, err := gorm.G[User](app.db).Where("email = ?", token.Claims.(*jwt.RegisteredClaims).Subject).First(c)
	if err != nil {
		fmt.Printf("User not found: %v\n", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Set(ContextUserKey, &user)

	// continue on to the next middleware / route handler
	c.Next()
}
