package auth

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Set a dummy JWT secret for testing
	os.Setenv("JWT_SECRET", "test-secret")
	// Run the tests
	exitCode := m.Run()
	// Clean up the environment variable
	os.Unsetenv("JWT_SECRET")
	os.Exit(exitCode)
}
