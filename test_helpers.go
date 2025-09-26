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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Test constants - centralized configuration
const (
	defaultTestEmail    = "test@test.com"
	defaultTestPassword = "secret123"
	defaultTestFileName = "testfile.txt"
	testFileContent     = "This is a test file content for testing purposes."
	testJWTSecret       = "test-jwt-secret-key-for-testing"
)

// setupTestApp creates a new isolated test app instance
// This ensures each test has a clean environment
func setupTestApp(t *testing.T) *App {
	t.Helper()

	// Set test environment variables
	require.NoError(t, os.Setenv("ENVIRONMENT", "TEST"))
	require.NoError(t, os.Setenv("JWT_SECRET", testJWTSecret))

	// Create fresh database and app instance
	db := setupDatabase()
	app := &App{db: db}
	app.router = app.setupRouter()

	return app
}

// cleanup removes test artifacts and performs cleanup
// Uses t.Helper() to improve test failure reporting
func cleanup(t *testing.T) {
	t.Helper()

	// Remove test database file
	if err := os.Remove("./test.db"); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: Failed to remove test.db: %v", err)
	}

	// Remove test files directory
	if err := os.RemoveAll("./files"); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: Failed to remove files directory: %v", err)
	}
}

// registerUser creates a test user and returns authentication cookie
// This is a common operation across many tests
func (app *App) registerUser(t *testing.T) *http.Cookie {
	t.Helper()

	user := User{Email: defaultTestEmail, Password: defaultTestPassword}
	userJSON, err := json.Marshal(user)
	require.NoError(t, err, "Failed to marshal test user")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/register", strings.NewReader(string(userJSON)))
	req.Header.Set("Content-Type", "application/json")
	app.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "User registration failed: %s", w.Body.String())

	cookies := w.Result().Cookies()
	require.NotEmpty(t, cookies, "No authentication cookie returned")

	return cookies[0]
}

// registerUserWithCredentials creates a user with specific credentials
func (app *App) registerUserWithCredentials(t *testing.T, email, password string) *http.Cookie {
	t.Helper()

	user := User{Email: email, Password: password}
	userJSON, err := json.Marshal(user)
	require.NoError(t, err, "Failed to marshal user with credentials")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/register", strings.NewReader(string(userJSON)))
	req.Header.Set("Content-Type", "application/json")
	app.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "User registration failed for %s: %s", email, w.Body.String())

	cookies := w.Result().Cookies()
	require.NotEmpty(t, cookies, "No authentication cookie returned for user %s", email)

	return cookies[0]
}

// createTestFile creates a temporary test file and ensures cleanup
func createTestFile(t *testing.T) string {
	t.Helper()

	err := os.WriteFile(defaultTestFileName, []byte(testFileContent), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Use t.Cleanup for automatic cleanup
	t.Cleanup(func() {
		if err := os.Remove(defaultTestFileName); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: Failed to cleanup test file %s: %v", defaultTestFileName, err)
		}
	})

	return defaultTestFileName
}

// createTag creates a test tag in the database
func (app *App) createTag(t *testing.T, name string) Tag {
	t.Helper()

	tag := Tag{Name: name}
	err := gorm.G[Tag](app.db).Create(context.Background(), &tag)
	require.NoError(t, err, "Failed to create test tag: %s", name)

	return tag
}

// FileUploadRequest represents a structured file upload request
type FileUploadRequest struct {
	Name        string
	Description string
	Tags        []uint
	FilePath    string
}

// uploadFile uploads a file with comprehensive error handling
func (app *App) uploadFile(t *testing.T, cookie *http.Cookie, req FileUploadRequest) *httptest.ResponseRecorder {
	t.Helper()

	// Validate required fields
	require.NotEmpty(t, req.Name, "File name is required")
	require.NotEmpty(t, req.FilePath, "File path is required")

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	require.NoError(t, writer.WriteField("name", req.Name))
	require.NoError(t, writer.WriteField("description", req.Description))

	// Add tags if provided
	if len(req.Tags) > 0 {
		tagStrings := make([]string, len(req.Tags))
		for i, tag := range req.Tags {
			tagStrings[i] = strconv.Itoa(int(tag))
		}
		require.NoError(t, writer.WriteField("tags", strings.Join(tagStrings, ",")))
	}

	// Add file
	file, err := os.Open(req.FilePath)
	require.NoError(t, err, "Failed to open test file: %s", req.FilePath)
	defer file.Close()

	part, err := writer.CreateFormFile("file", req.FilePath)
	require.NoError(t, err, "Failed to create form file part")

	_, err = io.Copy(part, file)
	require.NoError(t, err, "Failed to copy file content")

	require.NoError(t, writer.Close(), "Failed to close multipart writer")

	// Make request
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/files", body)
	httpReq.AddCookie(cookie)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	app.router.ServeHTTP(w, httpReq)

	return w
}

// getLatestFile retrieves the most recently created file with error handling
func (app *App) getLatestFile(t *testing.T) File {
	t.Helper()

	file, err := gorm.G[File](app.db).Preload("Tags", nil).Order("created_at desc").First(context.Background())
	require.NoError(t, err, "Failed to retrieve latest file from database")

	return file
}

// getFileByID retrieves a specific file by ID
func (app *App) getFileByID(t *testing.T, id uint) File {
	t.Helper()

	file, err := gorm.G[File](app.db).Preload("Tags", nil).Where("id = ?", id).First(context.Background())
	require.NoError(t, err, "Failed to retrieve file with ID %d", id)

	return file
}

// makeRequest is a helper for making HTTP requests with optional authentication
func (app *App) makeRequest(t *testing.T, method, path string, body io.Reader, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)

	// Set content type for JSON requests
	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json")
	}

	if cookie != nil {
		req.AddCookie(cookie)
	}

	app.router.ServeHTTP(w, req)

	return w
}

// makeRequestWithToken makes a request using JWT token authentication
func (app *App) makeRequestWithToken(t *testing.T, method, path string, body io.Reader, token string) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)

	// Set content type for JSON requests
	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.AddCookie(&http.Cookie{Name: "token", Value: token})
	}

	app.router.ServeHTTP(w, req)

	return w
}

// makeFormRequest makes a form-encoded request
func (app *App) makeFormRequest(t *testing.T, method, path string, formData string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if cookie != nil {
		req.AddCookie(cookie)
	}

	app.router.ServeHTTP(w, req)

	return w
}

// makeJSONRequest makes a JSON request with proper headers
func (app *App) makeJSONRequest(t *testing.T, method, path string, payload interface{}, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		require.NoError(t, err, "Failed to marshal JSON payload")
		body = strings.NewReader(string(jsonData))
	}

	return app.makeRequest(t, method, path, body, cookie)
}
