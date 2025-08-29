package main

import (
	"fmt"
	"os"

	"github.com/backend-project/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

type App struct {
	db *gorm.DB
}

func (app *App) register(c *gin.Context) {
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

func (app *App) login(c *gin.Context) {
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

func (app *App) logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
}

func (app *App) getFiles(c *gin.Context) {
}

func setupRouter() *gin.Engine {
	fmt.Println("Setting up router...")
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "PRODUCTION"
	}
	fmt.Printf("Running in %s\n", environment)

	var db *gorm.DB
	var err error
	if environment == "TEST" {
		fmt.Println("Using SQLite.")
		db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
	} else {
		fmt.Println("Using MySQL.")
		dsn := os.Getenv("DSN")
		if dsn == "" {
			panic("DSN environment variable not set.")
		}
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
	}

	// Migrate the schema
	err = db.AutoMigrate(&File{}, &Tag{}, &User{})
	if err != nil {
		panic("failed to run database migrations")
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	app := App{db: db}

	// auth
	router.POST("/register", app.register)
	router.POST("/login", app.login)
	router.GET("/logout", app.logout)

	// files crud
	router.GET("/files", app.getFiles)
	return router
}

func main() {
	router := setupRouter()

	fmt.Println("Running on localhost:8080")
	err := router.Run("localhost:8080")
	if err != nil {
		panic(err)
	}
}
