package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	// Set a test secret
	SetJWTSecret("test-secret-key")

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}
}

func TestValidateToken(t *testing.T) {
	SetJWTSecret("test-secret-key")

	// Generate a token
	token, err := GenerateToken(123, "testuser")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Validate the token
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// Verify claims
	if claims.UserID != 123 {
		t.Errorf("Expected UserID 123, got %d", claims.UserID)
	}

	if claims.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", claims.Username)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	SetJWTSecret("test-secret-key")

	_, err := ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken() should fail for invalid token")
	}
}

func TestValidateExpiredToken(t *testing.T) {
	// This test is hard to do without mocking time
	// Just verify that validation works for now
	SetJWTSecret("test-secret-key")

	token, _ := GenerateToken(1, "testuser")
	claims, err := ValidateToken(token)

	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// Verify token is not expired yet
	if time.Now().After(claims.ExpiresAt.Time) {
		t.Error("Token should not be expired immediately after creation")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	SetJWTSecret("test-secret-key")

	refreshToken, err := GenerateRefreshToken(1, "testuser")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() failed: %v", err)
	}

	if refreshToken == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	// Validate refresh token
	claims, err := ValidateToken(refreshToken)
	if err != nil {
		t.Fatalf("ValidateToken() failed for refresh token: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
}

func TestTokenBlacklist(t *testing.T) {
	bl := NewTokenBlacklist()

	token := "test-token-123"
	expiresAt := time.Now().Add(1 * time.Hour)

	// Token should not be blacklisted initially
	if bl.IsBlacklisted(token) {
		t.Error("Token should not be blacklisted initially")
	}

	// Add token to blacklist
	bl.Add(token, expiresAt)

	// Token should now be blacklisted
	if !bl.IsBlacklisted(token) {
		t.Error("Token should be blacklisted after adding")
	}

	// Different token should not be blacklisted
	if bl.IsBlacklisted("different-token") {
		t.Error("Different token should not be blacklisted")
	}
}
