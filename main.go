package main

import (
	"fmt"
	"github.com/backend-project/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var currentClaims jwt.Claims

func authMiddleware(c *gin.Context) {
	// find the jwt from cookies
	tokenString, err := c.Cookie("token")

	if err != nil {
		fmt.Println("JWT missing in cookies")
		//c.Redirect(http.StatusSeeOther, "/login")
		c.Abort()
		return
	}

	// verify jwt
	token, err := auth.VerifyJWT(tokenString)

	if err != nil {
		fmt.Printf("JWT verification failed: %v\n", err)
		//c.Redirect(http.StatusSeeOther, "/login")
		c.Abort()
		return
	}

	currentClaims = token.Claims

	fmt.Printf("JWT verified. Claims: %+v\\n", token.Claims)
	// continue on to the next middleware / route handler
	c.Next()
}

func register(c *gin.Context) {
	//var user models.User
	//
	//if err := c.BindJSON(&user); err != nil {
	//	c.AbortWithStatus(http.StatusBadRequest)
	//} else {
	//	// check if email already exists
	//	users := models.GetUsers()
	//
	//	if users != nil && len(users) > 0 {
	//		for _, foundUser := range users {
	//			if foundUser.Email == user.Email {
	//				c.AbortWithStatus(http.StatusFound)
	//				return
	//			}
	//		}
	//	}
	//
	//	// hash password
	//	hash, err := auth.HashPassword(user.Password)
	//
	//	if err != nil {
	//		c.AbortWithStatus(http.StatusInternalServerError)
	//		return
	//	}
	//
	//	user.Password = hash
	//
	//	models.Register(user)
	//	// generate JWT so we don't have to login again for 1 hour
	//	tokenString, err := auth.GenerateJWT(user.Email)
	//
	//	if err != nil {
	//		c.String(http.StatusInternalServerError, "Error creating JWT")
	//		return
	//	}
	//
	//	fmt.Printf("JWT created: %s\n", tokenString)
	//	c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
	//	// redirect to home page from login page
	//	//c.Redirect(http.StatusSeeOther, "/")
	//	c.IndentedJSON(http.StatusCreated, user)
	//}
}

func login(c *gin.Context) {
	//var user models.User
	//
	//if err := c.BindJSON(&user); err != nil {
	//	c.AbortWithStatus(http.StatusBadRequest)
	//} else {
	//	// check if email is in database
	//	emailExists, err := models.EmailExists(user.Email)
	//
	//	if err != nil {
	//		c.AbortWithStatus(http.StatusInternalServerError)
	//		return
	//	}
	//
	//	if !emailExists {
	//		c.String(http.StatusUnauthorized, "Invalid Credentials")
	//	} else {
	//		// check if password is correct
	//		hashedPassword, err := models.GetUserPassword(user.Email)
	//
	//		if err != nil {
	//			c.AbortWithStatus(http.StatusInternalServerError)
	//			return
	//		}
	//
	//		correctPassword := auth.CheckPasswordHash(user.Password, hashedPassword)
	//
	//		if !correctPassword {
	//			c.String(http.StatusUnauthorized, "Invalid Credentials")
	//			return
	//		} else {
	//			// generate JWT so we don't have to login again for 1 hour
	//			tokenString, err := auth.GenerateJWT(user.Email)
	//
	//			if err != nil {
	//				c.String(http.StatusInternalServerError, "Error creating JWT")
	//				return
	//			}
	//
	//			fmt.Printf("JWT created: %s\n", tokenString)
	//			c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
	//			// redirect to home page from login page
	//			//c.Redirect(http.StatusSeeOther, "/")
	//		}
	//	}
	//}
}

func logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
}
func main() {
	fmt.Println("Setting up router...")
	router := gin.Default()

	// auth
	router.POST("/register", register)
	router.POST("/login", login)
	router.GET("/logout", logout)

	fmt.Println("Running on localhost:8080")
	err := router.Run("localhost:8080")
	if err != nil {
		panic(err)
	}
}
