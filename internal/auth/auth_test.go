package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "my_secure_password"
	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error hashing password: %s", err)
	}

	match, err := CheckPasswordHash(password, hashedPassword)
	if err != nil {
		t.Fatalf("Error checking password hash: %s", err)
	}
	if !match {
		t.Fatal("Expected password to match hash, but it did not")
	}

	wrongPassword := "wrong_password"
	match, err = CheckPasswordHash(wrongPassword, hashedPassword)
	if err != nil {
		t.Fatalf("Error checking password hash: %s", err)
	}
	if match {
		t.Fatal("Expected wrong password to not match hash, but it did")
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "my_secret_key"
	expiresIn := time.Second

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	validatedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("Error validating JWT: %s", err)
	}
	if validatedUserID != userID {
		t.Fatalf("Expected validated user ID to be %s, but got %s", userID, validatedUserID)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "my_secret_key"
	expiresIn := -time.Second // Token that expires immediately

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Error making JWT: %s", err)
	}

	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatal("Expected error validating expired JWT, but got none")
	}
}

func TestGetBearerToken(t *testing.T) {
	header := http.Header{}
	header.Set("Authorization", "Bearer my_token")

	token, err := GetBearerToken(header)
	if err != nil {
		t.Fatalf("Error getting bearer token: %s", err)
	}
	if token != "my_token" {
		t.Fatalf("Expected token to be 'my_token', but got '%s'", token)
	}
}