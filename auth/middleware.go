package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(c *gin.Context) {
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

	fmt.Printf("JWT verified. Claims: %+v\\n", token.Claims)
	// continue on to the next middleware / route handler
	c.Next()
}
