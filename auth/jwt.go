package auth

import (
	"time"
	"os"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func getRole(email string) string {
	if email == "damien.z.hall@gmail.com" {
		return "admin"
	}

	return "default"
} 

func GenerateJWT(email string) (string, error) {
	err := godotenv.Load()

	if err != nil {
		return "", fmt.Errorf("could not load .env")
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": email,
		"iss": "snippet-app",
		"aud": getRole(email),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := claims.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {
	err := godotenv.Load()

	if err != nil {
		return nil, fmt.Errorf("could not load .env")
	}

	token, err := jwt.Parse(tokenString, func (token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}
