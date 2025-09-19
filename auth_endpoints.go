package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @Summary Register a new user
// @Description Register a new user with the given email and password
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   user     body    User   true        "User to register"
// @Success 201 {object} User
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /register [post]
func (app *App) register(c *gin.Context) {
	var user User

	if err := c.BindJSON(&user); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
	} else {
		// check if email already exists
		// TODO: change this to use .Where("email = ?", user.Email)
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
		hash, err := HashPassword(user.Password)

		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		user.Password = hash

		tx := app.db.Create(&user)
		if tx.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": tx.Error.Error()})
			return
		}
		// generate JWT so we don't have to login again for 1 hour
		tokenString, err := GenerateJWT(user.Email)

		if err != nil {
			c.String(http.StatusInternalServerError, "Error creating JWT")
			return
		}

		c.SetCookie("token", tokenString, 3600, "/", "localhost", false, true)
		// redirect to home page from login page
		//c.Redirect(http.StatusSeeOther, "/")

		// TODO: There must be a better way of doing this, just don't want to return the hash
		user.Password = ""

		c.JSON(http.StatusCreated, user)
	}
}

// @Summary Log in a user
// @Description Log in a user with the given email and password
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   user     body    User   true        "User to log in"
// @Success 200 {object} object
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /login [post]
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
			} else {
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
}

// @Summary Log out a user
// @Description Log out a user by clearing the token cookie
// @Tags auth
// @Success 200
// @Router /logout [get]
func (app *App) logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "localhost", false, true)
}
