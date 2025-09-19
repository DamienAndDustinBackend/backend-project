package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// @Summary Get a list of tags for the authenticated user
// @Description Get a list of tags for the authenticated user. The user is identified by the token in the cookie.
// @Tags tags
// @Produce  json
// @Success 200 {array} Tag
// @Failure 404
// @Router /tags [get]
func (app *App) getTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	var tags []Tag
	result := app.db.Where(&Tag{UserId: user.ID}).Find(&tags)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Could not find tag(s)"})
		return
	}

	c.JSON(http.StatusFound, tags)
	return
}

// @Summary Create new tags
// @Description Create new tags for the authenticated user. The user is identified by the token in the cookie.
// @Tags tags
// @Accept  multipart/form-data
// @Produce  json
// @Param tagnames formData string true "Comma-separated list of tag names to create"
// @Success 201 {array} Tag
// @Failure 400
// @Failure 404
// @Router /tags [post]
func (app *App) createTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	// , is now an unsupported char in tags
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)
	var newTags []Tag

	for _, name := range tagNames {
		var tag Tag
		app.db.Find(&tag, "name = ?", name)

		if tag.Name != name {
			newTags = append(newTags, Tag{
				Name:   name,
				UserId: user.ID,
			})
		}
	}

	result := app.db.Create(newTags)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Could not bulk create tags!",
		})

		return
	}

	for _, tag := range newTags {
		result := app.db.First(&tag, "name = ?", tag.Name)

		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": result.Error})
			return
		}
	}

	c.JSON(http.StatusCreated, newTags)
}

// @Summary Edit existing tags
// @Description Edit existing tags for the authenticated user. The user is identified by the token in the cookie.
// @Tags tags
// @Accept  multipart/form-data
// @Produce  json
// @Param tagnames formData string true "Comma-separated list of tag names to edit"
// @Param newnames formData string true "Comma-separated list of new tag names"
// @Success 200 {string} string "Tags modified!"
// @Failure 412
// @Failure 404
// @Failure 500
// @Router /tags [put]
func (app *App) editTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)
	newNames := strings.Split(c.PostForm("newnames"), `,`)

	if len(tagNames) != len(newNames) {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "Tag Names and New Names length don't match!"})
	}

	for i, name := range tagNames {
		var tag Tag
		result := app.db.Where(&Tag{UserId: user.ID, Name: name}).First(&tag)

		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Could not find tag '%s'!", name),
			})

			return
		}

		tag.Name = newNames[i]
		result = app.db.Save(&tag)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Could not save tag '%s' to the DB!", name),
			})

			return
		}
	}

	c.JSON(http.StatusOK, "Tags modified!")
}

// @Summary Delete existing tags
// @Description Delete existing tags for the authenticated user. The user is identified by the token in the cookie.
// @Tags tags
// @Accept  multipart/form-data
// @Produce  json
// @Param tagnames formData string true "Comma-separated list of tag names to delete"
// @Success 200 {string} string "Tags deleted!"
// @Failure 404
// @Failure 400
// @Router /tags [delete]
func (app *App) deleteTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)

	for _, name := range tagNames {
		result := app.db.Where(&Tag{UserId: user.ID, Name: name}).Delete(&Tag{})
		if result.RowsAffected < 1 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Could not delete tag '%s' as it does not exist!", name),
			})

			return
		}
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Could not delete tag '%s'!", name),
			})

			return
		}
	}

	c.JSON(http.StatusOK, "Tags deleted!")
}
