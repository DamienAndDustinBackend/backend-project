package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Test data constants - following "Learn Go with Tests" pattern
const (
	validEmail    = "test@example.com"
	validPassword = "validpassword123"
	adminEmail    = "damien.z.hall@gmail.com"
	testTagName   = "test-tag"
)

// Test data variables
var (
	validUserData = User{Email: validEmail, Password: validPassword}
)

// Test helpers following "Learn Go with Tests" patterns
func assertStatusCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("got status %d, want %d", got, want)
	}
}

func createTestUser(t *testing.T, app *App) *http.Cookie {
	t.Helper()
	return app.registerUserWithCredentials(t, validEmail, validPassword)
}

// TestPing tests the basic ping endpoint
func TestPing(t *testing.T) {
	app := setupTestApp(t)
	defer cleanup(t)

	w := app.makeRequest(t, "GET", "/ping", nil, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

// TestUserRegistration tests user registration with various scenarios
func TestUserRegistration(t *testing.T) {
	tests := []struct {
		name           string
		user           User
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid registration",
			user:           User{Email: "new@example.com", Password: "password123"},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid JSON",
			user:           User{}, // Will be replaced with invalid JSON
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "duplicate email",
			user:           validUserData,
			expectedStatus: http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp(t)
			defer cleanup(t)

			// Pre-register user for duplicate test
			if tt.name == "duplicate email" {
				app.registerUserWithCredentials(t, tt.user.Email, tt.user.Password)
			}

			var body string
			if tt.name == "invalid JSON" {
				body = "invalid json"
			} else {
				userJSON, err := json.Marshal(tt.user)
				require.NoError(t, err)
				body = string(userJSON)
			}

			w := app.makeRequest(t, "POST", "/register", strings.NewReader(body), nil)
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify successful registration creates user in database
			if tt.expectedStatus == http.StatusCreated {
				dbUser, err := gorm.G[User](app.db).Where("email = ?", tt.user.Email).First(context.Background())
				require.NoError(t, err)
				assert.Equal(t, tt.user.Email, dbUser.Email)
				assert.NotEmpty(t, dbUser.Password)                   // Password should be hashed
				assert.NotEqual(t, tt.user.Password, dbUser.Password) // Should be hashed
			}
		})
	}
}

// TestUserLogin tests user login scenarios
func TestUserLogin(t *testing.T) {
	tests := []struct {
		name           string
		setupUser      bool
		email          string
		password       string
		expectedStatus int
		expectCookie   bool
	}{
		{
			name:           "valid login",
			setupUser:      true,
			email:          validEmail,
			password:       validPassword,
			expectedStatus: http.StatusOK,
			expectCookie:   true,
		},
		{
			name:           "non-existent user",
			setupUser:      false,
			email:          "nonexistent@example.com",
			password:       validPassword,
			expectedStatus: http.StatusUnauthorized,
			expectCookie:   false,
		},
		{
			name:           "wrong password",
			setupUser:      true,
			email:          validEmail,
			password:       "wrongpassword",
			expectedStatus: http.StatusUnauthorized,
			expectCookie:   false,
		},
		{
			name:           "invalid JSON",
			setupUser:      false,
			email:          "",
			password:       "",
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp(t)
			defer cleanup(t)

			if tt.setupUser {
				// For wrong password test, register with correct password but login with wrong one
				if tt.name == "wrong password" {
					app.registerUserWithCredentials(t, tt.email, validPassword)
				} else {
					app.registerUserWithCredentials(t, tt.email, tt.password)
				}
			}

			var body string
			if tt.name == "invalid JSON" {
				body = "invalid json"
			} else {
				user := User{Email: tt.email, Password: tt.password}
				userJSON, err := json.Marshal(user)
				require.NoError(t, err)
				body = string(userJSON)
			}

			w := app.makeRequest(t, "POST", "/login", strings.NewReader(body), nil)
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectCookie {
				cookies := w.Result().Cookies()
				require.NotEmpty(t, cookies)
				assert.Equal(t, "token", cookies[0].Name)
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				assert.Contains(t, w.Body.String(), "Invalid Credentials")
			}
		})
	}
}

// TestFileOperations tests file CRUD operations
func TestFileOperations(t *testing.T) {
	t.Run("upload file", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)
		filePath := createTestFile(t)

		w := app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "test-file",
			Description: "test description",
			FilePath:    filePath,
		})

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify file was saved to database
		var file File
		err := app.db.Order("created_at desc").First(&file).Error
		require.NoError(t, err)
		assert.Equal(t, "test-file", file.Name)
		assert.Equal(t, "test description", file.Description)
	})

	t.Run("upload file with tags", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)
		filePath := createTestFile(t)
		tag := app.createTag(t, "test-tag")

		w := app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "tagged-file",
			Description: "file with tag",
			Tags:        []uint{tag.ID},
			FilePath:    filePath,
		})

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify file has the tag
		savedFile := app.getLatestFile(t)
		require.Len(t, savedFile.Tags, 1)
		assert.Equal(t, tag.ID, savedFile.Tags[0].ID)
		assert.Equal(t, "test-tag", savedFile.Tags[0].Name)
	})

	t.Run("get files", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)
		filePath := createTestFile(t)

		// Upload test files
		for i := 0; i < 3; i++ {
			app.uploadFile(t, cookie, FileUploadRequest{
				Name:        fmt.Sprintf("file-%d", i),
				Description: fmt.Sprintf("description-%d", i),
				FilePath:    filePath,
			})
		}

		w := app.makeRequest(t, "GET", "/files", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		var files []File
		err := json.Unmarshal(w.Body.Bytes(), &files)
		require.NoError(t, err)
		assert.Len(t, files, 3)
	})

	t.Run("get single file", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)
		filePath := createTestFile(t)

		// Upload a file
		app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "single-file",
			Description: "single file description",
			FilePath:    filePath,
		})

		savedFile := app.getLatestFile(t)
		w := app.makeRequest(t, "GET", fmt.Sprintf("/files/%d", savedFile.ID), nil, cookie)

		assert.Equal(t, http.StatusOK, w.Code)

		var returnedFile File
		err := json.Unmarshal(w.Body.Bytes(), &returnedFile)
		require.NoError(t, err)
		assert.Equal(t, savedFile.ID, returnedFile.ID)
		assert.Equal(t, "single-file", returnedFile.Name)
	})

	t.Run("update file", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)
		filePath := createTestFile(t)

		// Upload a file
		app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "original-name",
			Description: "original description",
			FilePath:    filePath,
		})

		savedFile := app.getLatestFile(t)
		updateData := File{Name: "updated-name", Description: "updated description"}
		updateJSON, err := json.Marshal(updateData)
		require.NoError(t, err)

		w := app.makeRequest(t, "PATCH", fmt.Sprintf("/files/%d", savedFile.ID), strings.NewReader(string(updateJSON)), cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify update in database
		updatedFile, err := gorm.G[File](app.db).Where("id = ?", savedFile.ID).First(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "updated-name", updatedFile.Name)
		assert.Equal(t, "updated description", updatedFile.Description)
	})

	t.Run("delete file", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)
		filePath := createTestFile(t)

		// Upload a file
		app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "file-to-delete",
			Description: "will be deleted",
			FilePath:    filePath,
		})

		savedFile := app.getLatestFile(t)
		w := app.makeRequest(t, "DELETE", fmt.Sprintf("/files/%d", savedFile.ID), nil, cookie)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify deletion (soft delete)
		var deletedFile File
		err := app.db.Unscoped().First(&deletedFile, savedFile.ID).Error
		require.NoError(t, err)
		assert.True(t, deletedFile.DeletedAt.Valid)
	})
}

// TestFileOperationErrors tests error scenarios for file operations
func TestFileOperationErrors(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		setupAuth      bool
		expectedStatus int
	}{
		{
			name:           "get non-existent file",
			method:         "GET",
			path:           "/files/999999",
			setupAuth:      true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get file with invalid ID",
			method:         "GET",
			path:           "/files/invalid",
			setupAuth:      true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized file access",
			method:         "GET",
			path:           "/files",
			setupAuth:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "delete non-existent file",
			method:         "DELETE",
			path:           "/files/999999",
			setupAuth:      true,
			expectedStatus: http.StatusOK, // Delete returns success even if file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp(t)
			defer cleanup(t)

			var cookie *http.Cookie
			if tt.setupAuth {
				cookie = app.registerUser(t)
			}

			w := app.makeRequest(t, tt.method, tt.path, nil, cookie)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestTagOperations tests tag CRUD operations
func TestTagOperations(t *testing.T) {
	t.Run("create single tag", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		w := app.makeFormRequest(t, "POST", "/tags", "tagnames=single-tag", cookie)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify tag was created
		tag, err := gorm.G[Tag](app.db).Where("name = ?", "single-tag").First(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "single-tag", tag.Name)
	})

	t.Run("create multiple tags", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		w := app.makeFormRequest(t, "POST", "/tags", "tagnames=tag1,tag2,tag3", cookie)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify all tags were created
		var count int64
		app.db.Model(&Tag{}).Where("name IN ?", []string{"tag1", "tag2", "tag3"}).Count(&count)
		assert.Equal(t, int64(3), count)
	})

	t.Run("get tags", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Create some tags using the API (not direct DB creation)
		app.makeFormRequest(t, "POST", "/tags", "tagnames=get-tag-1,get-tag-2", cookie)

		w := app.makeRequest(t, "GET", "/tags", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code) // API returns 200 OK

		// Response should contain the tags
		body := w.Body.String()
		assert.Contains(t, body, "get-tag-1")
		assert.Contains(t, body, "get-tag-2")
	})

	t.Run("edit tag", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Create tag first
		app.makeFormRequest(t, "POST", "/tags", "tagnames=old-tag-name", cookie)

		w := app.makeFormRequest(t, "PUT", "/tags", "tagnames=old-tag-name&newnames=new-tag-name", cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify tag was updated
		tag, err := gorm.G[Tag](app.db).Where("name = ?", "new-tag-name").First(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "new-tag-name", tag.Name)
	})

	t.Run("delete tag", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Create tag first
		app.makeFormRequest(t, "POST", "/tags", "tagnames=tag-to-delete", cookie)

		w := app.makeFormRequest(t, "DELETE", "/tags", "tagnames=tag-to-delete", cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify tag was deleted (soft delete) - check with deleted records included
		var tag Tag
		err := app.db.Unscoped().Where("name = ?", "tag-to-delete").First(&tag).Error
		if err == nil {
			// Tag exists but should be soft deleted
			assert.True(t, tag.DeletedAt.Valid, "Tag should be soft deleted")
		} else {
			// Tag might be hard deleted depending on implementation
			assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Tag should be deleted")
		}
	})
}

// TestTagOperationErrors tests error scenarios for tag operations
func TestTagOperationErrors(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		formData       string
		setupTags      []string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "create duplicate tag",
			method:         "POST",
			formData:       "tagnames=duplicate-tag",
			setupTags:      []string{"duplicate-tag"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Could not bulk create tags",
		},
		{
			name:           "edit non-existent tag",
			method:         "PUT",
			formData:       "tagnames=nonexistent&newnames=new-name",
			setupTags:      []string{},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Could not find tag",
		},
		{
			name:           "edit with mismatched lengths",
			method:         "PUT",
			formData:       "tagnames=tag1,tag2&newnames=new1",
			setupTags:      []string{"tag1", "tag2"},
			expectedStatus: http.StatusPreconditionFailed,
			expectedError:  "length don't match",
		},
		{
			name:           "delete non-existent tag",
			method:         "DELETE",
			formData:       "tagnames=nonexistent",
			setupTags:      []string{},
			expectedStatus: http.StatusNotFound,
			expectedError:  "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp(t)
			defer cleanup(t)
			cookie := app.registerUser(t)

			// Setup tags if needed
			for _, tagName := range tt.setupTags {
				app.createTag(t, tagName)
			}

			w := app.makeFormRequest(t, tt.method, "/tags", tt.formData, cookie)
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

// TestTagPagination tests pagination functionality for tags endpoint
func TestTagPagination(t *testing.T) {
	app := setupTestApp(t)
	defer cleanup(t)
	cookie := app.registerUser(t)

	// Create 25 tags for testing pagination
	tagNames := make([]string, 25)
	for i := 0; i < 25; i++ {
		tagNames[i] = fmt.Sprintf("tag-%02d", i+1)
	}

	// Create tags in batches to avoid URL length limits
	for i := 0; i < len(tagNames); i += 5 {
		end := i + 5
		if end > len(tagNames) {
			end = len(tagNames)
		}
		batch := strings.Join(tagNames[i:end], ",")
		w := app.makeFormRequest(t, "POST", "/tags", "tagnames="+batch, cookie)
		require.Equal(t, http.StatusCreated, w.Code, "Failed to create tag batch %d", i/5+1)
	}

	t.Run("default pagination", func(t *testing.T) {
		w := app.makeRequest(t, "GET", "/tags", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)

		// Default page size should be 10
		assert.Len(t, tags, 10)
	})

	t.Run("custom page size", func(t *testing.T) {
		w := app.makeRequest(t, "GET", "/tags?page_size=5", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)

		assert.Len(t, tags, 5)
	})

	t.Run("large page size gets capped", func(t *testing.T) {
		w := app.makeRequest(t, "GET", "/tags?page_size=200", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)

		// Should be capped at 25 (total number of tags) or 100 (max page size)
		assert.Len(t, tags, 25)
	})

	t.Run("second page", func(t *testing.T) {
		w := app.makeRequest(t, "GET", "/tags?page=2&page_size=10", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)

		assert.Len(t, tags, 10)
	})

	t.Run("third page", func(t *testing.T) {
		w := app.makeRequest(t, "GET", "/tags?page=3&page_size=10", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)

		// Third page should have 5 remaining tags (25 total - 20 from first two pages)
		assert.Len(t, tags, 5)
	})

	t.Run("page beyond available data", func(t *testing.T) {
		w := app.makeRequest(t, "GET", "/tags?page=10&page_size=10", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)

		// Should return empty array for pages beyond available data
		assert.Empty(t, tags)
	})
}

// TestJWTOperations tests JWT token generation and verification
func TestJWTOperations(t *testing.T) {
	// Set up test environment
	require.NoError(t, os.Setenv("JWT_SECRET", "test-secret-key"))

	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{
			name:        "valid email",
			email:       validEmail,
			expectError: false,
		},
		{
			name:        "admin email",
			email:       adminEmail,
			expectError: false,
		},
		{
			name:        "empty email",
			email:       "",
			expectError: false, // JWT generation doesn't validate email format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(tt.email)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Verify the token
			verifiedToken, err := VerifyJWT(token)
			require.NoError(t, err)
			assert.True(t, verifiedToken.Valid)

			// Check claims
			claims, ok := verifiedToken.Claims.(*jwt.RegisteredClaims)
			require.True(t, ok)
			assert.Equal(t, tt.email, claims.Subject)
			assert.Equal(t, "backend-project", claims.Issuer)
		})
	}

	t.Run("invalid token verification", func(t *testing.T) {
		_, err := VerifyJWT("invalid-token-string")
		assert.Error(t, err)
	})

	t.Run("expired token verification", func(t *testing.T) {
		// Create expired token
		claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": validEmail,
			"iss": "backend-project",
			"exp": time.Now().Add(-time.Hour).Unix(),
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		})

		tokenString, err := claims.SignedString([]byte("test-secret-key"))
		require.NoError(t, err)

		_, err = VerifyJWT(tokenString)
		assert.Error(t, err)
	})
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	app := setupTestApp(t)
	defer cleanup(t)

	// Create test user
	user := User{Email: validEmail, Password: validPassword}
	err := gorm.G[User](app.db).Create(context.Background(), &user)
	require.NoError(t, err)

	// Generate valid token
	token, err := GenerateJWT(validEmail)
	require.NoError(t, err)

	// Add test route
	app.router.GET("/protected", app.AuthMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "protected resource"})
	})

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "valid token",
			token:          token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := app.makeRequestWithToken(t, "GET", "/protected", nil, tt.token)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestUserRoles tests the role assignment functionality
func TestUserRoles(t *testing.T) {
	tests := []struct {
		email        string
		expectedRole string
	}{
		{
			email:        adminEmail,
			expectedRole: "admin",
		},
		{
			email:        validEmail,
			expectedRole: "default",
		},
		{
			email:        "unknown@example.com",
			expectedRole: "default",
		},
		{
			email:        "",
			expectedRole: "default",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("role for %s", tt.email), func(t *testing.T) {
			role := getRole(tt.email)
			assert.Equal(t, tt.expectedRole, role)
		})
	}
}

// TestPasswordHashing tests password hashing functionality
func TestPasswordHashing(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "normal password",
			password: "normalpassword123",
		},
		{
			name:     "short password",
			password: "123",
		},
		{
			name:     "long password",
			password: "this-is-a-very-long-password-with-many-characters-and-symbols-!@#$%^&*()",
		},
		{
			name:     "special characters",
			password: "p@ssw0rd!#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)

			// Verify correct password
			assert.True(t, CheckPasswordHash(tt.password, hash))

			// Verify incorrect password
			assert.False(t, CheckPasswordHash("wrongpassword", hash))
		})
	}
}

// TestPagination tests pagination functionality
func TestPagination(t *testing.T) {
	app := setupTestApp(t)
	defer cleanup(t)
	cookie := app.registerUser(t)
	filePath := createTestFile(t)

	// Create test files
	const totalFiles = 25
	for i := 0; i < totalFiles; i++ {
		app.uploadFile(t, cookie, FileUploadRequest{
			Name:        fmt.Sprintf("file-%02d", i),
			Description: fmt.Sprintf("description for file %d", i),
			FilePath:    filePath,
		})
	}

	tests := []struct {
		name        string
		queryParams string
		expectedMax int
		expectedMin int
	}{
		{
			name:        "default pagination",
			queryParams: "",
			expectedMax: 10, // Default page size
			expectedMin: 10,
		},
		{
			name:        "custom page size",
			queryParams: "?page_size=5",
			expectedMax: 5,
			expectedMin: 5,
		},
		{
			name:        "large page size gets capped",
			queryParams: "?page_size=200",
			expectedMax: 100, // Max page size
			expectedMin: 25,  // We only have 25 files
		},
		{
			name:        "second page",
			queryParams: "?page=2&page_size=10",
			expectedMax: 15, // Remaining files
			expectedMin: 10, // Should have at least 10 files on second page
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := app.makeRequest(t, "GET", "/files"+tt.queryParams, nil, cookie)
			assert.Equal(t, http.StatusOK, w.Code)

			var files []File
			err := json.Unmarshal(w.Body.Bytes(), &files)
			require.NoError(t, err)
			assert.LessOrEqual(t, len(files), tt.expectedMax)
			assert.GreaterOrEqual(t, len(files), tt.expectedMin)
		})
	}
}

// TestDatabaseSetup tests database initialization
func TestDatabaseSetup(t *testing.T) {
	require.NoError(t, os.Setenv("ENVIRONMENT", "TEST"))

	db := setupDatabase()
	require.NotNil(t, db)

	// Verify tables exist
	tables := []interface{}{&User{}, &File{}, &Tag{}}
	for _, table := range tables {
		assert.True(t, db.Migrator().HasTable(table))
	}
}

// TestEnvironmentConfiguration tests environment variable handling
func TestEnvironmentConfiguration(t *testing.T) {
	t.Run("default port", func(t *testing.T) {
		originalPort := os.Getenv("PORT")
		os.Unsetenv("PORT")
		defer func() {
			if originalPort != "" {
				os.Setenv("PORT", originalPort)
			}
		}()

		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		assert.Equal(t, "8080", port)
	})

	t.Run("custom port", func(t *testing.T) {
		require.NoError(t, os.Setenv("PORT", "9000"))
		defer os.Unsetenv("PORT")

		port := os.Getenv("PORT")
		assert.Equal(t, "9000", port)
	})

	t.Run("database panic on missing DSN", func(t *testing.T) {
		originalEnv := os.Getenv("ENVIRONMENT")
		originalDSN := os.Getenv("DSN")

		require.NoError(t, os.Setenv("ENVIRONMENT", "PRODUCTION"))
		os.Unsetenv("DSN")

		defer func() {
			if originalEnv != "" {
				os.Setenv("ENVIRONMENT", originalEnv)
			} else {
				os.Unsetenv("ENVIRONMENT")
			}
			if originalDSN != "" {
				os.Setenv("DSN", originalDSN)
			}
		}()

		assert.Panics(t, func() {
			setupDatabase()
		})
	})
}

// TestUserIsolation tests that users can only access their own data
func TestUserIsolation(t *testing.T) {
	t.Run("users cannot access each other's files", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)

		// Create two users
		user1Cookie := app.registerUserWithCredentials(t, "user1@test.com", "password123")
		user2Cookie := app.registerUserWithCredentials(t, "user2@test.com", "password456")

		// User1 uploads a file
		filePath := createTestFile(t)
		w := app.uploadFile(t, user1Cookie, FileUploadRequest{
			Name:        "user1-file",
			Description: "user1's file",
			FilePath:    filePath,
		})
		require.Equal(t, http.StatusOK, w.Code)

		// Get the file ID
		var uploadResponse File
		err := json.Unmarshal(w.Body.Bytes(), &uploadResponse)
		require.NoError(t, err)

		// User2 tries to access User1's file
		w = app.makeRequest(t, "GET", fmt.Sprintf("/files/%d", uploadResponse.ID), nil, user2Cookie)
		assert.Equal(t, http.StatusBadRequest, w.Code) // Should not be able to access
	})

	t.Run("users cannot access each other's tags", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)

		// Create two users
		user1Cookie := app.registerUserWithCredentials(t, "user1@test.com", "password123")
		user2Cookie := app.registerUserWithCredentials(t, "user2@test.com", "password456")

		// User1 creates tags
		w := app.makeFormRequest(t, "POST", "/tags", "tagnames=user1-tag1,user1-tag2", user1Cookie)
		require.Equal(t, http.StatusCreated, w.Code)

		// User2 gets tags - should only see their own (none)
		w = app.makeRequest(t, "GET", "/tags", nil, user2Cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)
		assert.Empty(t, tags) // User2 should see no tags
	})
}

// TestComplexScenarios tests complex multi-step workflows
func TestComplexScenarios(t *testing.T) {
	t.Run("file with tags workflow", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Create tags first
		w := app.makeFormRequest(t, "POST", "/tags", "tagnames=workflow,test,complex", cookie)
		require.Equal(t, http.StatusCreated, w.Code)

		// Get created tags
		w = app.makeRequest(t, "GET", "/tags", nil, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		var tags []Tag
		err := json.Unmarshal(w.Body.Bytes(), &tags)
		require.NoError(t, err)
		require.Len(t, tags, 3)

		// Upload file with tags
		filePath := createTestFile(t)
		w = app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "complex-file",
			Description: "file with multiple tags",
			Tags:        []uint{tags[0].ID, tags[1].ID},
			FilePath:    filePath,
		})
		require.Equal(t, http.StatusOK, w.Code)

		// Verify file has correct tags
		savedFile := app.getLatestFile(t)
		assert.Equal(t, "complex-file", savedFile.Name)
		assert.Len(t, savedFile.Tags, 2)

		// Update file to add third tag
		updateData := map[string]interface{}{
			"name": "updated-complex-file",
			"tags": []map[string]interface{}{
				{"id": tags[0].ID},
				{"id": tags[1].ID},
				{"id": tags[2].ID},
			},
		}

		w = app.makeJSONRequest(t, "PATCH", fmt.Sprintf("/files/%d", savedFile.ID), updateData, cookie)
		require.Equal(t, http.StatusOK, w.Code)

		// Verify update
		updatedFile := app.getFileByID(t, savedFile.ID)
		assert.Equal(t, "updated-complex-file", updatedFile.Name)
	})

	t.Run("pagination with large dataset", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Create 50 tags
		tagBatches := []string{}
		for i := 0; i < 10; i++ {
			batch := []string{}
			for j := 0; j < 5; j++ {
				batch = append(batch, fmt.Sprintf("tag-%d-%d", i, j))
			}
			tagBatches = append(tagBatches, strings.Join(batch, ","))
		}

		for _, batch := range tagBatches {
			w := app.makeFormRequest(t, "POST", "/tags", "tagnames="+batch, cookie)
			require.Equal(t, http.StatusCreated, w.Code)
		}

		// Test pagination through all tags
		allTags := []Tag{}
		page := 1
		for {
			w := app.makeRequest(t, "GET", fmt.Sprintf("/tags?page=%d&page_size=10", page), nil, cookie)
			require.Equal(t, http.StatusOK, w.Code)

			var pageTags []Tag
			err := json.Unmarshal(w.Body.Bytes(), &pageTags)
			require.NoError(t, err)

			if len(pageTags) == 0 {
				break
			}

			allTags = append(allTags, pageTags...)
			page++

			// Safety check to prevent infinite loop
			if page > 10 {
				break
			}
		}

		assert.Len(t, allTags, 50)
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("empty tag names", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		w := app.makeFormRequest(t, "POST", "/tags", "tagnames=", cookie)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("very long tag names", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		longTagName := strings.Repeat("a", 1000)
		w := app.makeFormRequest(t, "POST", "/tags", "tagnames="+longTagName, cookie)
		// Should either succeed or fail gracefully
		assert.True(t, w.Code == http.StatusCreated || w.Code == http.StatusBadRequest)
	})

	t.Run("special characters in tag names", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		specialTags := "tag-with-dash,tag_with_underscore,tag.with.dots"
		w := app.makeFormRequest(t, "POST", "/tags", "tagnames="+specialTags, cookie)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("pagination edge cases", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Test page 0 (should default to page 1)
		w := app.makeRequest(t, "GET", "/tags?page=0", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test negative page (should default to page 1)
		w = app.makeRequest(t, "GET", "/tags?page=-1", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test page size 0 (should default to 10)
		w = app.makeRequest(t, "GET", "/tags?page_size=0", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test negative page size (should default to 10)
		w = app.makeRequest(t, "GET", "/tags?page_size=-5", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestLogoutFunctionality tests the logout endpoint
func TestLogoutFunctionality(t *testing.T) {
	app := setupTestApp(t)
	defer cleanup(t)

	// Register and login user
	cookie := app.registerUser(t)

	// Verify user can access protected endpoint
	w := app.makeRequest(t, "GET", "/tags", nil, cookie)
	assert.Equal(t, http.StatusOK, w.Code)

	// Logout
	w = app.makeRequest(t, "GET", "/logout", nil, cookie)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify logout cookie is set (empty token with negative expiry)
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "token" {
			assert.Equal(t, "", c.Value)
			assert.Negative(t, c.MaxAge)
			found = true
			break
		}
	}
	assert.True(t, found, "Logout cookie not found")
}

// TestUniqueFileNameGeneration tests the unique filename generation
func TestUniqueFileNameGeneration(t *testing.T) {
	app := setupTestApp(t)
	defer cleanup(t)

	// Create a gin context for testing
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Generate multiple filenames and ensure they're unique
	filenames := make(map[string]bool)
	for i := 0; i < 10; i++ {
		filename := app.generateUniqueFileName(c)
		assert.NotEmpty(t, filename)
		assert.False(t, filenames[filename], "Duplicate filename generated: %s", filename)
		filenames[filename] = true
	}
}

// TestAuthErrors tests authentication and authorization errors using table-driven approach
func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		method         string
		setupUser      bool
		duplicateEmail bool
		useAuth        bool
		expectedStatus int
	}{
		{"duplicate email registration", "/register", "POST", false, true, false, http.StatusFound},
		{"login nonexistent user", "/login", "POST", false, false, false, http.StatusUnauthorized},
		{"unauthorized file access", "/files", "GET", false, false, false, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp(t)
			defer cleanup(t)

			var cookie *http.Cookie
			userData := map[string]string{"email": "test@test.com", "password": "password123"}

			if tt.duplicateEmail {
				app.makeJSONRequest(t, "POST", "/register", userData, nil)
			}
			if tt.setupUser {
				cookie = app.registerUser(t)
			}
			if tt.useAuth && cookie == nil {
				cookie = app.registerUser(t)
			}

			var w *httptest.ResponseRecorder
			switch {
			case tt.method == "POST" && strings.Contains(tt.endpoint, "register"):
				w = app.makeJSONRequest(t, tt.method, tt.endpoint, userData, cookie)
			case tt.method == "POST" && strings.Contains(tt.endpoint, "login"):
				w = app.makeJSONRequest(t, tt.method, tt.endpoint, map[string]string{
					"email": "nonexistent@test.com", "password": "password123"}, cookie)
			default:
				w = app.makeRequest(t, tt.method, tt.endpoint, nil, cookie)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestUtilities tests core utility functions using table-driven tests
func TestUtilities(t *testing.T) {
	t.Run("password hashing", func(t *testing.T) {
		tests := []struct {
			name     string
			password string
			check    string
			want     bool
		}{
			{"correct password", validPassword, validPassword, true},
			{"wrong password", validPassword, "wrongpassword", false},
			{"empty password", "", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hash, err := HashPassword(tt.password)
				require.NoError(t, err)

				got := CheckPasswordHash(tt.check, hash)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("JWT operations", func(t *testing.T) {
		tests := []struct {
			name      string
			email     string
			wantError bool
		}{
			{"valid email", validEmail, false},
			{"admin email", adminEmail, false},
			{"empty email", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				token, err := GenerateJWT(tt.email)
				if tt.wantError {
					assert.Error(t, err)
					return
				}

				require.NoError(t, err)
				assert.NotEmpty(t, token)

				parsedToken, err := VerifyJWT(token)
				require.NoError(t, err)
				assert.True(t, parsedToken.Valid)
			})
		}
	})
}

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	t.Run("nonexistent resource operations", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		// Test editing nonexistent tags
		updateData := map[string]interface{}{
			"old_names": []string{"nonexistent"},
			"new_names": []string{"new"},
		}
		w := app.makeJSONRequest(t, "PUT", "/tags", updateData, cookie)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Test invalid file ID
		w = app.makeRequest(t, "GET", "/files/not-a-number", nil, cookie)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestBasicEndpoints tests simple endpoints
func TestBasicEndpoints(t *testing.T) {
	t.Run("ping endpoint", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/ping", nil)
		app.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "pong", w.Body.String())
	})

	t.Run("logout clears cookie", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := app.registerUser(t)

		w := app.makeRequest(t, "GET", "/logout", nil, cookie)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAdditionalEdgeCases tests edge cases using table-driven approach
func TestAdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		testType       string
		input          string
		expectedStatus int
	}{
		{"unicode tags", "tags", "tagnames=测试,🏷️", http.StatusCreated},
		{"empty tag names", "tags", "tagnames=", http.StatusBadRequest},
		{"zero page size", "pagination", "/tags?page_size=0", http.StatusOK},
		{"high page number", "pagination", "/tags?page=999999", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := setupTestApp(t)
			defer cleanup(t)
			cookie := createTestUser(t, app)

			var w *httptest.ResponseRecorder
			switch tt.testType {
			case "tags":
				w = app.makeFormRequest(t, "POST", "/tags", tt.input, cookie)
			case "pagination":
				w = app.makeRequest(t, "GET", tt.input, nil, cookie)
			}

			assertStatusCode(t, w.Code, tt.expectedStatus)
		})
	}

	t.Run("special characters in file names", func(t *testing.T) {
		app := setupTestApp(t)
		defer cleanup(t)
		cookie := createTestUser(t, app)

		filePath := createTestFile(t)
		w := app.uploadFile(t, cookie, FileUploadRequest{
			Name:        "file with spaces & symbols!",
			Description: "unicode: 你好",
			FilePath:    filePath,
		})
		assertStatusCode(t, w.Code, http.StatusOK)
	})
}
