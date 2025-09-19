package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type App struct {
	db *gorm.DB
}

func (app *App) setupRouter() *gin.Engine {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load .env file")
	}

	fmt.Println("Setting up router...")

	router := gin.Default()
	router.MaxMultipartMemory = 10 * 1_073_741_824 // 10 GiB

	// https://gin-gonic.com/en/docs/deployment/#dont-trust-all-proxies
	err = router.SetTrustedProxies(nil)
	if err != nil {
		panic(err)
	}

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	// auth
	router.POST("/register", app.register)
	router.POST("/login", app.login)
	router.GET("/logout", app.logout)

	// files
	authorized := router.Group("/")
	authorized.Use(app.AuthMiddleware)
	authorized.GET("/files/:id", app.getFile)
	authorized.GET("/files", app.getFiles)
	authorized.POST("/files", app.createFile)
	authorized.PATCH("/files/:id", app.updateFile)
	authorized.DELETE("/files/:id", app.deleteFile)
	return router
}

func setupDatabase() *gorm.DB {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Failed to load .env file")
	}

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
		fmt.Println("Using Postgres.")
		dsn := os.Getenv("DSN")
		if dsn == "" {
			panic("DSN environment variable not set.")
		}
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
	}

	// Migrate the schema
	err = db.AutoMigrate(&User{}, &File{}, &Tag{})
	if err != nil {
		panic("failed to run database migrations")
	}

	return db
}

func main() {
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	err := router.Run(":" + port)
	if err != nil {
		panic(err)
	}
}
