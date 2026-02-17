package service

import (
	"os"
	"strconv"
	"time"

	"github.com/braccet/auth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID      uint64 `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"name"`
	jwt.RegisteredClaims
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	return []byte(secret)
}

func getJWTExpiry() time.Duration {
	expiry := os.Getenv("JWT_EXPIRY")
	if expiry == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(expiry)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// IssueToken creates a new JWT for the given user
func IssueToken(user *domain.User) (string, error) {
	claims := &Claims{
		UserID:      user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(getJWTExpiry())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ValidateToken parses and validates a JWT, returning the claims
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
