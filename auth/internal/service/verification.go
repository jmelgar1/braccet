package service

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"
)

const (
	// TokenBytes is the number of random bytes for token generation (32 bytes = 256 bits)
	TokenBytes = 32
	// DefaultTokenExpiry is how long verification tokens are valid
	DefaultTokenExpiry = 1 * time.Hour
)

// GenerateVerificationToken creates a cryptographically secure random token
func GenerateVerificationToken() (string, error) {
	bytes := make([]byte, TokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetTokenExpiry returns the configured token expiration duration
func GetTokenExpiry() time.Duration {
	if expiry := os.Getenv("VERIFICATION_TOKEN_EXPIRY"); expiry != "" {
		if d, err := time.ParseDuration(expiry); err == nil {
			return d
		}
	}
	return DefaultTokenExpiry
}

// BuildVerificationURL constructs the full verification URL
func BuildVerificationURL(baseURL, token string) string {
	return baseURL + "/auth/verify-email?token=" + token
}
