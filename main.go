package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/backend-project/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/joho/godotenv"
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
	var user User
	
	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
	} else {
		// check if email already exists
		var users []User
		result := app.db.Find(&users)

		if result.Error == nil && len(users) > 0 {
			for _, foundUser := range users {
				if foundUser.Email == user.Email {
					c.AbortWithStatus(http.StatusFound)
					return
				}
			}
		}
	
		// hash password
		hash, err := auth.HashPassword(user.Password)
	
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	
		user.Password = hash
	

		app.db.Create(&user)
		// generate JWT so we don't have to login again for 1 hour
		tokenString, err := auth.GenerateJWT(user.Email)
	
		if err != nil {
			c.String(http.StatusInternalServerError, "Error creating JWT")
			return
		}
	
		fmt.Printf("JWT created: %s\n", tokenString)
		c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
		// redirect to home page from login page
		//c.Redirect(http.StatusSeeOther, "/")
		c.IndentedJSON(http.StatusCreated, user)
	}
}

func (app *App) login(c *gin.Context) {
	var user User
	
	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
	} else {
		// check if email is in database
		var user User
		result := app.db.Where("email = ?", user.Email)
	
		if result.Error != nil {
			c.String(http.StatusUnauthorized, "Invalid Credentials")
		} else {
			// check if password is correct
			hashedPassword := user.Password
	
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
	
			correctPassword := auth.CheckPasswordHash(user.Password, hashedPassword)
	
			if !correctPassword {
				c.String(http.StatusUnauthorized, "Invalid Credentials")
				return
			} else {
				// generate JWT so we don't have to login again for 1 hour
				tokenString, err := auth.GenerateJWT(user.Email)
	
				if err != nil {
					c.String(http.StatusInternalServerError, "Error creating JWT")
					return
				}
	
				fmt.Printf("JWT created: %s\n", tokenString)
				c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
				// redirect to home page from login page
				//c.Redirect(http.StatusSeeOther, "/")
			}
		}
	}
}

func (app *App) logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
}

func (app *App) getFiles(c *gin.Context) {
}

func (app *App) doesFileNameExist(ctx *gin.Context, fileName string) bool {
	_, err := gorm.G[File](app.db).Where("file_path LIKE ?", fileName).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		panic(err)
	} else {
		return true
	}
}

func (app *App) generateUniqueFileName(ctx *gin.Context) string {
	uniqueName := uuid.New().String()

	// Timestamp?

	// I'm not sure if this is necessary, but I'm not sure if the uuid is really guaranteed to be unique
	for app.doesFileNameExist(ctx, uniqueName) {
		uniqueName = uuid.New().String()
	}

	return uniqueName
}

func (app *App) createFile(c *gin.Context) {

	uploadedFile, err := c.FormFile("file")
	fileName := c.PostForm("name")
	fileDescription := c.DefaultPostForm("description", "")
	//tags := c.DefaultPostForm("tags", "[]")
	//tagStructs := []Tag{}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uploadPath := os.Getenv("UPLOAD_PATH")
	if uploadPath == "" {
		uploadPath = "./files"
	}

	uniqueFileName := filepath.Base(app.generateUniqueFileName(c))

	err = c.SaveUploadedFile(uploadedFile, fmt.Sprintf("%s/%s", uploadPath, uniqueFileName))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file := File{Name: fileName, Description: fileDescription, FilePath: uniqueFileName, Tags: []Tag{}}
	err = gorm.G[File](app.db).Create(
		c,
		&file,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// I'm querying the database here to get the updatedAt, createdAt, timestamps
	fileFromDatabase, err := gorm.G[File](app.db).Where(&File{ID: file.ID}).First(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fileFromDatabase)
}

func (app *App) setupRouter() *gin.Engine {
func setupRouter() *gin.Engine {
	godotenv.Load()
	fmt.Println("Setting up router...")

	router := gin.Default()
	router.MaxMultipartMemory = 10 * 1_073_741_824 // 10 GiB

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	// auth
	router.POST("/register", app.register)
	router.POST("/login", app.login)
	router.GET("/logout", app.logout)

	// files crud
	router.GET("/files", app.getFiles)
	router.POST("/files", app.createFile)
	return router
}

func setupDatabase() *gorm.DB {
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

	return db
}

func main() {
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	fmt.Println("Running on localhost:8080")
	err := router.Run("localhost:8080")
	if err != nil {
		panic(err)
	}
}
