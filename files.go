package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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
	var files []File
	result := app.db.Scopes(Paginate(c.Request)).Find(&files)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
	}
	c.JSON(http.StatusOK, files)
	return
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

func (app *App) getFile(c *gin.Context) {
	fileId := c.Param("id")
	file, err := gorm.G[File](app.db).Where("id = ?", fileId).First(c)

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
	fileId := c.Param("id")
	_, err := gorm.G[File](app.db).Where("id = ?", fileId).Delete(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (app *App) updateFile(c *gin.Context) {
	fileId := c.Param("id")

	var file File
	if err := c.BindJSON(&file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := gorm.G[File](app.db).Where("id = ?", fileId).Updates(c, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
