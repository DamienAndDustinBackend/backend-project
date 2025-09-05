package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func cleanUp() {
	err := os.Remove("./test.db")
	if err != nil {
		panic(err)
	}

	err = os.RemoveAll("./files")
	if err != nil {
		panic(err)
	}
}

func TestPingRoute(t *testing.T) {
	defer cleanUp()

	err := os.Setenv("ENVIRONMENT", "TEST")
	if err != nil {
		panic(err)
	}
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

func TestUploadFileRoute(t *testing.T) {
	defer cleanUp()

	err := os.Setenv("ENVIRONMENT", "TEST")
	if err != nil {
		panic(err)
	}

	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	// Create a dummy file for testing
	dummyFileContent := []byte("This is a test file content.")
	dummyFileName := "testfile.txt"
	err = os.WriteFile(dummyFileName, dummyFileContent, 0644)
	assert.NoError(t, err)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			panic(err)
		}
	}(dummyFileName) // Clean up the dummy file

	// Create a new multipart writer
	fileBody := new(bytes.Buffer)
	writer := multipart.NewWriter(fileBody)

	// Create a form file field
	file, err := os.Open(dummyFileName)
	assert.NoError(t, err)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	_, err = writer.CreateFormField("name")
	if err != nil {
		return
	}
	err = writer.WriteField("name", "test-filename")
	if err != nil {
		return
	}

	_, err = writer.CreateFormField("description")
	if err != nil {
		return
	}
	err = writer.WriteField("description", "this is a test file")
	if err != nil {
		return
	}

	_, err = writer.CreateFormField("tags")
	if err != nil {
		return
	}
	err = writer.WriteField("description", "[]")
	if err != nil {
		return
	}

	part, err := writer.CreateFormFile("file", dummyFileName) // "myFile" must match the name in the handler

	assert.NoError(t, err)
	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	// Close the multipart writer
	err = writer.Close()
	assert.NoError(t, err)

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("POST", "/files", fileBody)
	req.Header.Set("Content-Type", writer.FormDataContentType()) // Set the correct Content-Type header
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	expected, err := gorm.G[File](app.db).Order("created_at desc").First(context.TODO())
	if err != nil {
		panic(err)
	}
	expectedJson, err := json.Marshal(expected)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, string(expectedJson), w.Body.String())
}

func TestGetFiles(t *testing.T) {
	defer cleanUp()

	err := os.Setenv("ENVIRONMENT", "TEST")
	if err != nil {
		panic(err)
	}
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/files", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "[]", w.Body.String())

	// upload a file
	// Create a dummy file for testing
	dummyFileContent := []byte("This is a test file content.")
	dummyFileName := "testfile.txt"
	err = os.WriteFile(dummyFileName, dummyFileContent, 0644)
	assert.NoError(t, err)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			panic(err)
		}
	}(dummyFileName) // Clean up the dummy file

	// Create a new multipart writer
	fileBody := new(bytes.Buffer)
	writer := multipart.NewWriter(fileBody)

	// Create a form file field
	file, err := os.Open(dummyFileName)
	assert.NoError(t, err)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	_, err = writer.CreateFormField("name")
	if err != nil {
		return
	}
	err = writer.WriteField("name", "test-filename")
	if err != nil {
		return
	}

	_, err = writer.CreateFormField("description")
	if err != nil {
		return
	}
	err = writer.WriteField("description", "this is a test file")
	if err != nil {
		return
	}

	_, err = writer.CreateFormField("tags")
	if err != nil {
		return
	}
	err = writer.WriteField("description", "[]")
	if err != nil {
		return
	}

	part, err := writer.CreateFormFile("file", dummyFileName) // "myFile" must match the name in the handler

	assert.NoError(t, err)
	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	// Close the multipart writer
	err = writer.Close()
	assert.NoError(t, err)

	w = httptest.NewRecorder()

	req, _ = http.NewRequest("POST", "/files", fileBody)
	req.Header.Set("Content-Type", writer.FormDataContentType()) // Set the correct Content-Type header
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	expected, err := gorm.G[File](app.db).Order("created_at desc").First(context.TODO())
	if err != nil {
		panic(err)
	}
	expectedJson, err := json.Marshal(expected)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, string(expectedJson), w.Body.String())

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/files", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	expectedJson, err = json.Marshal([]File{expected})
	assert.Equal(t, string(expectedJson), w.Body.String())
}
