package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (app *App) register(c *gin.Context) {
	var user User

	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// check if email already exists
	_, err := gorm.G[User](app.db).Where("email = ?", user.Email).First(c)
	if err == nil {
		// User with this email already exists
		c.AbortWithStatus(http.StatusFound)
		return
	}

	// hash password
	hash, err := HashPassword(user.Password)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	user.Password = hash

	err = gorm.G[User](app.db).Create(c, &user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// generate JWT so we don't have to login again for 1 hour
	tokenString, err := GenerateJWT(user.Email)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating JWT")
		return
	}

	c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)

	// TODO: There must be a better way of doing this, just don't want to return the hash
	user.Password = ""

	c.JSON(http.StatusCreated, user)
}

func (app *App) login(c *gin.Context) {
	var user User

	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
	} else {
		// check if email is in database
		databaseUser, err := gorm.G[User](app.db).Where("email = ?", user.Email).First(c)

		if err != nil {
			c.String(http.StatusUnauthorized, "Invalid Credentials")
		} else {
			// check if password is correct
			hashedPassword := databaseUser.Password

			correctPassword := CheckPasswordHash(user.Password, hashedPassword)

			if !correctPassword {
				c.String(http.StatusUnauthorized, "Invalid Credentials")
				return
			}

			// generate JWT so we don't have to login again for 1 hour
			tokenString, err := GenerateJWT(user.Email)

			if err != nil {
				c.String(http.StatusInternalServerError, "Error creating JWT")
				return
			}

			c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
			c.JSON(http.StatusOK, gin.H{"success": true})
		}
	}
}

func (app *App) logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
}
