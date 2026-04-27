package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "securepassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
	if hash == password {
		t.Fatal("HashPassword returned plaintext password")
	}
	// Should start with argon2id prefix
	if len(hash) < 10 || hash[:9] != "$argon2id" {
		t.Fatalf("Hash doesn't have argon2id prefix: %s", hash[:20])
	}
}

func TestHashPasswordUniqueSalts(t *testing.T) {
	hash1, _ := HashPassword("samepassword")
	hash2, _ := HashPassword("samepassword")
	if hash1 == hash2 {
		t.Fatal("Two hashes of the same password should differ (different salts)")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "mypassword456"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !valid {
		t.Fatal("VerifyPassword should return true for correct password")
	}
}

func TestVerifyPasswordWrong(t *testing.T) {
	hash, _ := HashPassword("correctpassword")
	valid, err := VerifyPassword("wrongpassword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if valid {
		t.Fatal("VerifyPassword should return false for wrong password")
	}
}

func TestVerifyPasswordInvalidHash(t *testing.T) {
	_, err := VerifyPassword("password", "not-a-valid-hash")
	if err == nil {
		t.Fatal("VerifyPassword should return error for invalid hash format")
	}
}

func TestVerifyPasswordEmpty(t *testing.T) {
	hash, _ := HashPassword("")
	valid, err := VerifyPassword("", hash)
	if err != nil {
		t.Fatalf("VerifyPassword with empty password failed: %v", err)
	}
	if !valid {
		t.Fatal("Empty password should verify against its own hash")
	}

	valid, _ = VerifyPassword("notempty", hash)
	if valid {
		t.Fatal("Non-empty password should not verify against empty password hash")
	}
}
