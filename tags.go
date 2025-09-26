package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getTagNames extracts tag names from a slice of tags
func getTagNames(tags []Tag) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	return names
}

func (app *App) getTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)

	// Use traditional API for Scopes since generic API doesn't support it yet
	var tags []Tag
	result := app.db.Scopes(Paginate(c.Request)).Where(&Tag{UserID: user.ID}).Find(&tags)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Could not find tag(s)"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

func (app *App) createTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	// , is now an unsupported char in tags
	tagNamesRaw := c.PostForm("tagnames")
	if strings.TrimSpace(tagNamesRaw) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tag names cannot be empty",
		})
		return
	}

	tagNames := strings.Split(tagNamesRaw, `,`)
	var newTags []Tag

	for _, name := range tagNames {
		// Skip empty tag names
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Check if tag already exists
		_, err := gorm.G[Tag](app.db).Where("name = ?", name).First(c)
		if err != nil {
			// Tag doesn't exist, add it to newTags
			newTags = append(newTags, Tag{
				Name:   name,
				UserID: user.ID,
			})
		}
	}

	if len(newTags) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Could not bulk create tags!",
		})
		return
	}

	// Create tags individually using generic API
	for i := range newTags {
		err := gorm.G[Tag](app.db).Create(c, &newTags[i])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Could not bulk create tags!",
			})
			return
		}
	}

	// Fetch the created tags with their IDs
	createdTags, err := gorm.G[Tag](app.db).Where("user_id = ? AND name IN ?", user.ID, getTagNames(newTags)).Find(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdTags)
}

func (app *App) editTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)
	newNames := strings.Split(c.PostForm("newnames"), `,`)

	if len(tagNames) != len(newNames) {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "Tag Names and New Names length don't match!"})
		return
	}

	for i, name := range tagNames {
		tag, err := gorm.G[Tag](app.db).Where(&Tag{UserID: user.ID, Name: name}).First(c)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Could not find tag '%s'!", name),
			})
			return
		}

		tag.Name = newNames[i]
		_, err = gorm.G[Tag](app.db).Updates(c, tag)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Could not save tag '%s' to the DB!", name),
			})
			return
		}
	}

	c.JSON(http.StatusOK, "Tags modified!")
}

func (app *App) deleteTags(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)
	tagNames := strings.Split(c.PostForm("tagnames"), `,`)

	for _, name := range tagNames {
		rowsAffected, err := gorm.G[Tag](app.db).Where(&Tag{UserID: user.ID, Name: name}).Delete(c)
		if rowsAffected < 1 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Could not delete tag '%s' as it does not exist!", name),
			})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Could not delete tag '%s'!", name),
			})
			return
		}
	}

	c.JSON(http.StatusOK, "Tags deleted!")
}
