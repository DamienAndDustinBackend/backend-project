package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (app *App) getTags(c *gin.Context) {
	var tags []Tag
	result := app.db.Find(&tags)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, tags)
	return
}

func (app *App) createTags(c *gin.Context) {
	// , is now an unsupported char in tags
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)
	var newTags []Tag

	for _, name := range tagNames {
		newTags = append(newTags, Tag{
			Name: name,
		})
	}

	result := app.db.Create(newTags)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, "Tags created!")
}

func (app *App) editTags(c *gin.Context) {
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)
	newNames := strings.Split(c.PostForm("newnames"), `,`)

	if len(tagNames) != len(newNames) {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "Tag Names and New Names length don't match!"})
	}

	for i, name := range tagNames {
		var tag Tag
		result := app.db.First(&tag, "name = ?", name)

		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": result.Error})
			return
		}

		tag.Name = newNames[i]
		result = app.db.Save(&tag)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error})
			return
		}
	}

	c.JSON(http.StatusOK, "Tags modified!")
}

func (app *App) deleteTags(c *gin.Context) {
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)

	for _, name := range tagNames {
		result := app.db.Delete(&Tag{}, "Name LIKE ?", name)
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
			return
		}
	}

	c.JSON(http.StatusOK, "Tags deleted!")
}
