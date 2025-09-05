package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestGetFile(t *testing.T) {
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
	req, _ = http.NewRequest("GET", fmt.Sprintf("/files/%s", strconv.Itoa(int(expected.ID))), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, string(expectedJson), w.Body.String())
}

func TestDeleteFile(t *testing.T) {
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
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/files/%s", strconv.Itoa(int(expected.ID))), nil)
	router.ServeHTTP(w, req)

	expectedJson, err = json.Marshal(gin.H{"success": true})

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, string(expectedJson), w.Body.String())

	_, err = gorm.G[File](app.db).Where("id = ?", expected.ID).First(context.TODO())
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestUpdateFile(t *testing.T) {
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

	updatedFile := File{Name: "new name", Description: "new description"}
	updatedFileJson, _ := json.Marshal(updatedFile)
	req, _ = http.NewRequest("PATCH", fmt.Sprintf("/files/%s", strconv.Itoa(int(expected.ID))), strings.NewReader(string(updatedFileJson)))
	router.ServeHTTP(w, req)

	expectedJson, err = json.Marshal(gin.H{"success": true})

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, string(expectedJson), w.Body.String())

	fetchedUpdatedFile, err := gorm.G[File](app.db).Where("id = ?", expected.ID).First(context.TODO())
	if err != nil {
		panic(err)
	}
	assert.EqualValues(t, fetchedUpdatedFile.Name, updatedFile.Name)
	assert.EqualValues(t, fetchedUpdatedFile.Description, updatedFile.Description)
}

func TestRegister(t *testing.T) {
	defer cleanUp()

	err := os.Setenv("ENVIRONMENT", "TEST")
	if err != nil {
		panic(err)
	}
	err = os.Setenv("JWT_SECRET", "very-secret")
	if err != nil {
		panic(err)
	}
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	w := httptest.NewRecorder()

	user := User{Email: "test@test.com", Password: "secret"}
	userJson, _ := json.Marshal(user)
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(string(userJson)))
	router.ServeHTTP(w, req)

	fetchedUser, err := gorm.G[User](app.db).Where("email = ?", user.Email).First(context.TODO())
	if err != nil {
		panic(err)
	}

	assert.Equal(t, http.StatusCreated, w.Code)

	var returnedUser User
	err = json.Unmarshal(w.Body.Bytes(), &returnedUser)
	fmt.Println(user)
	if err != nil {
		return
	}
	assert.Equal(t, user.Email, fetchedUser.Email)
	assert.Equal(t, user.Email, returnedUser.Email)
	assert.Equal(t, "token", w.Result().Cookies()[0].Name)
}

func TestLogin(t *testing.T) {
	defer cleanUp()

	err := os.Setenv("ENVIRONMENT", "TEST")
	if err != nil {
		panic(err)
	}
	err = os.Setenv("JWT_SECRET", "very-secret")
	if err != nil {
		panic(err)
	}
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	// create a user (have to hit the endpoint, so the password gets hashed)
	w := httptest.NewRecorder()
	user := User{Email: "test@test.com", Password: "secret"}
	userJson, _ := json.Marshal(user)
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(string(userJson)))
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()

	req, _ = http.NewRequest("POST", "/login", strings.NewReader(string(userJson)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{\"success\":true}", w.Body.String())
	assert.Equal(t, "token", w.Result().Cookies()[0].Name)
}

func TestLogout(t *testing.T) {
	defer cleanUp()

	err := os.Setenv("ENVIRONMENT", "TEST")
	if err != nil {
		panic(err)
	}
	err = os.Setenv("JWT_SECRET", "very-secret")
	if err != nil {
		panic(err)
	}
	db := setupDatabase()
	app := App{db: db}
	router := app.setupRouter()

	// create a user (have to hit the endpoint, so the password gets hashed)
	w := httptest.NewRecorder()
	user := User{Email: "test@test.com", Password: "secret"}
	userJson, _ := json.Marshal(user)
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(string(userJson)))
	router.ServeHTTP(w, req)
	assert.Equal(t, "token", w.Result().Cookies()[0].Name)

	cookie := w.Result().Cookies()[0]
	w = httptest.NewRecorder()

	req, _ = http.NewRequest("POST", "/logout", strings.NewReader(string(userJson)))
	req.AddCookie(cookie)
	router.ServeHTTP(w, req)

	assert.Len(t, w.Result().Cookies(), 0)
}
