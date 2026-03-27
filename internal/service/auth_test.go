package service_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/service"
)

func TestAuthService_GenerateAndValidateTokenPair(t *testing.T) {
	svc := service.NewAuthService("test-secret-key-for-jwt")
	userID := uuid.New()

	pair, err := svc.GenerateTokenPair(userID)
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Error("refresh token should not be empty")
	}
	if pair.ExpiresAt == 0 {
		t.Error("expires_at should not be zero")
	}

	// Validate access token
	got, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if got != userID {
		t.Errorf("access token user = %s, want %s", got, userID)
	}

	// Validate refresh token
	got, err = svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("validate refresh token: %v", err)
	}
	if got != userID {
		t.Errorf("refresh token user = %s, want %s", got, userID)
	}
}

func TestAuthService_AccessTokenRejectsRefresh(t *testing.T) {
	svc := service.NewAuthService("test-secret-key")
	userID := uuid.New()

	pair, err := svc.GenerateTokenPair(userID)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Access token validation should reject refresh token
	if _, err := svc.ValidateAccessToken(pair.RefreshToken); err == nil {
		t.Error("should reject refresh token as access token")
	}

	// Refresh token validation should reject access token
	if _, err := svc.ValidateRefreshToken(pair.AccessToken); err == nil {
		t.Error("should reject access token as refresh token")
	}
}

func TestAuthService_InvalidToken(t *testing.T) {
	svc := service.NewAuthService("test-secret-key")

	if _, err := svc.ValidateAccessToken("invalid.token.here"); err == nil {
		t.Error("should reject invalid token")
	}
}

func TestAuthService_WrongSecret(t *testing.T) {
	svc1 := service.NewAuthService("secret-one")
	svc2 := service.NewAuthService("secret-two")

	pair, err := svc1.GenerateTokenPair(uuid.New())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := svc2.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Error("should reject token signed with different secret")
	}
}

func TestHashPassword_And_CheckPassword(t *testing.T) {
	password := "my-secure-password"

	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if hash == password {
		t.Error("hash should not equal plaintext")
	}

	if !service.CheckPassword(hash, password) {
		t.Error("correct password should match hash")
	}

	if service.CheckPassword(hash, "wrong-password") {
		t.Error("wrong password should not match hash")
	}
}
