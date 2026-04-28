package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWT configuration
var (
	tokenTTL = time.Hour * 24 * 7 // 7 days

	secretOnce  sync.Once
	cachedSecret []byte
)

// getJWTSecret returns the symmetric key used to sign + verify JWTs.
// Resolution order:
//
//  1. JWT_SECRET env var — for dev/CI overrides
//  2. <data-dir>/jwt.secret — generated and persisted on first run.
//     This is the .app distribution path: every install gets its own
//     random 32-byte secret, so two LocalFinance users can't decode
//     each other's tokens even if they reverse the binary.
//
// File mode is 0600 so other macOS users on the machine can't read it.
//
// Falls back to an in-memory random secret if writing fails (so the
// app boots in unusual environments) — at the cost of invalidating
// existing sessions whenever the process restarts.
func getJWTSecret() []byte {
	secretOnce.Do(func() {
		if env := os.Getenv("JWT_SECRET"); env != "" {
			cachedSecret = []byte(env)
			return
		}
		path := filepath.Join(dataDirForSecret(), "jwt.secret")
		if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
			cachedSecret = data
			return
		}
		// First run (or unreadable file). Generate and persist.
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			// crypto/rand failed — fall back to a process-lifetime random
			// secret so we don't ship the old hardcoded value.
			slog.Error("jwt: crypto/rand failed; using ephemeral secret", "err", err)
			raw = []byte(time.Now().Format(time.RFC3339Nano))
		}
		// Write hex-encoded so the file is grep-friendly and copy-pastable.
		hexBytes := []byte(hex.EncodeToString(raw))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
			if err := os.WriteFile(path, hexBytes, 0o600); err != nil {
				slog.Warn("jwt: could not persist secret; will be regenerated next run", "path", path, "err", err)
			}
		}
		cachedSecret = hexBytes
	})
	return cachedSecret
}

// dataDirForSecret resolves the directory where jwt.secret lives.
// Honors LOCALFINANCE_DATA_DIR (matches main.go's dataDir helper) and
// falls back to ~/Library/Application Support/LocalFinance.
func dataDirForSecret() string {
	if d := os.Getenv("LOCALFINANCE_DATA_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// No home — last-ditch /tmp so the app at least boots.
		return fmt.Sprintf("/tmp/localfinance-%d", os.Getuid())
	}
	return filepath.Join(home, "Library", "Application Support", "LocalFinance")
}

// Claims represents the JWT claims
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user
func GenerateToken(userID uuid.UUID, email string) (string, int64, error) {
	expirationTime := time.Now().Add(tokenTTL)
	
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "localfinance-thesaurus",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(getJWTSecret())
	if err != nil {
		return "", 0, err
	}

	return tokenString, expirationTime.Unix(), nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// RefreshToken generates a new token for a user if the current token is valid
func RefreshToken(tokenString string) (string, int64, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return "", 0, err
	}

	// Check if token is close to expiration (within 24 hours)
	if time.Until(claims.ExpiresAt.Time) > time.Hour*24 {
		return "", 0, errors.New("token doesn't need refresh yet")
	}

	return GenerateToken(claims.UserID, claims.Email)
}

// ExtractTokenFromHeader extracts Bearer token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", errors.New("invalid authorization header format")
	}
	return authHeader[7:], nil
}