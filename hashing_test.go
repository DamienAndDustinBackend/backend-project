package main

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Errorf("HashPassword() error = %v, wantErr %v", err, false)
	}

	if len(hash) == 0 {
		t.Errorf("HashPassword() hash is empty")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "mysecretpassword"
	hash, _ := HashPassword(password)

	// Test with correct password
	if !CheckPasswordHash(password, hash) {
		t.Errorf("CheckPasswordHash() = %v, want %v", false, true)
	}

	// Test with incorrect password
	if CheckPasswordHash("wrongpassword", hash) {
		t.Errorf("CheckPasswordHash() = %v, want %v", true, false)
	}
}
