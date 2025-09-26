package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Paginate https://gorm.io/docs/scopes.html#Pagination
func Paginate(r *http.Request) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		if page <= 0 {
			page = 1
		}

		pageSize, _ := strconv.Atoi(q.Get("page_size"))
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func (app *App) getFiles(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)

	// Use traditional API for Scopes since generic API doesn't support it yet
	var files []File
	result := app.db.Scopes(Paginate(c.Request)).Preload("Tags", nil).Where(&File{UserID: user.ID}).Find(&files)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, files)
}

func (app *App) doesFileNameExist(ctx *gin.Context, fileName string) bool {
	_, err := gorm.G[File](app.db).Where("file_path LIKE ?", fileName).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		panic(err)
	}
	return true
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

	user := c.MustGet(ContextUserKey).(*User)

	uploadedFile, err := c.FormFile("file")
	fileName := c.PostForm("name")
	fileDescription := c.DefaultPostForm("description", "")

	// Parse tags
	tagsString := c.DefaultPostForm("tags", "")
	var parsedTags []Tag
	if tagsString != "" {
		tagIDs := strings.Split(tagsString, `,`)

		for _, id := range tagIDs {
			uintID32, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				panic(err)
			}
			parsedTags = append(parsedTags, Tag{
				ID: uint(uintID32),
			})
		}
	}

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

	file := File{Name: fileName, Description: fileDescription, FilePath: uniqueFileName, UserID: user.ID}
	err = gorm.G[File](app.db).Create(
		c,
		&file,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Associate tags with the file
	if len(parsedTags) > 0 {
		// You need to use the classic GORM API for associations. ???
		err = app.db.Model(&file).Association("Tags").Append(parsedTags)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// I'm querying the database here to get the updatedAt and createdAt timestamps
	fileFromDatabase, err := gorm.G[File](app.db).Preload("Tags", nil).Where("id = ?", file.ID).First(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fileFromDatabase)
}

func (app *App) getFile(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)

	// TODO: There must be a better way
	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fileIDAsUint := uint(fileID)

	file, err := gorm.G[File](app.db).Preload("Tags", nil).Where(&File{UserID: user.ID, ID: fileIDAsUint}).First(c)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, file)
}

func (app *App) deleteFile(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)

	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileIDAsUint := uint(fileID)

	_, err = gorm.G[File](app.db).Where(&File{UserID: user.ID, ID: fileIDAsUint}).Delete(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (app *App) updateFile(c *gin.Context) {
	user := c.MustGet(ContextUserKey).(*User)

	fileID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fileIDAsUint := uint(fileID)

	var file File
	if err := c.BindJSON(&file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = gorm.G[File](app.db).Where(&File{UserID: user.ID, ID: fileIDAsUint}).Updates(c, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fetchedFile, err := gorm.G[File](app.db).Where(&File{UserID: user.ID, ID: fileIDAsUint}).First(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Associate tags with the file
	if len(file.Tags) > 0 {
		// You need to use the classic GORM API for associations. ??? maybe not
		fmt.Println(file.Tags)
		err = app.db.Model(&fetchedFile).Association("Tags").Replace(file.Tags)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
