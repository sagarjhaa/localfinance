package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateToken(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"

	token, expiresAt, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}
	if expiresAt == 0 {
		t.Fatal("GenerateToken returned zero expiration")
	}
}

func TestValidateToken(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"

	token, _, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Fatalf("Expected email %s, got %s", email, claims.Email)
	}
}

func TestValidateTokenInvalid(t *testing.T) {
	_, err := ValidateToken("invalid.token.here")
	if err == nil {
		t.Fatal("ValidateToken should fail for invalid token")
	}
}

func TestValidateTokenEmpty(t *testing.T) {
	_, err := ValidateToken("")
	if err == nil {
		t.Fatal("ValidateToken should fail for empty token")
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		header  string
		want    string
		wantErr bool
	}{
		{"Bearer abc123", "abc123", false},
		{"Bearer eyJhbGciOiJIUzI1NiJ9.test.sig", "eyJhbGciOiJIUzI1NiJ9.test.sig", false},
		{"bearer abc123", "", true},  // case sensitive
		{"abc123", "", true},          // no Bearer prefix
		{"", "", true},                // empty
		{"Bear", "", true},            // too short
	}

	for _, tt := range tests {
		got, err := ExtractTokenFromHeader(tt.header)
		if (err != nil) != tt.wantErr {
			t.Errorf("ExtractTokenFromHeader(%q) error = %v, wantErr %v", tt.header, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ExtractTokenFromHeader(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestRefreshTokenTooEarly(t *testing.T) {
	userID := uuid.New()
	token, _, _ := GenerateToken(userID, "test@example.com")

	// Token was just created (7 day TTL), so refresh should fail (>24h until expiry)
	_, _, err := RefreshToken(token)
	if err == nil {
		t.Fatal("RefreshToken should fail when token doesn't need refresh yet")
	}
}
