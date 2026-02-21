package service

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/braccet/auth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	UserID      uint64 `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"name"`
	TokenType   string `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair contains both access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	return []byte(secret)
}

func getAccessTokenExpiry() time.Duration {
	expiry := os.Getenv("ACCESS_TOKEN_EXPIRY")
	if expiry == "" {
		return 15 * time.Minute
	}
	d, err := time.ParseDuration(expiry)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func getRefreshTokenExpiry() time.Duration {
	expiry := os.Getenv("REFRESH_TOKEN_EXPIRY")
	if expiry == "" {
		return 7 * 24 * time.Hour // 7 days
	}
	d, err := time.ParseDuration(expiry)
	if err != nil {
		return 7 * 24 * time.Hour
	}
	return d
}

// IssueAccessToken creates a new access JWT for the given user
func IssueAccessToken(user *domain.User) (string, error) {
	claims := &Claims{
		UserID:      user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		TokenType:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(getAccessTokenExpiry())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// IssueRefreshToken creates a new refresh JWT for the given user
func IssueRefreshToken(user *domain.User) (string, error) {
	claims := &Claims{
		UserID:      user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		TokenType:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(getRefreshTokenExpiry())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// IssueTokenPair creates both access and refresh tokens for the user
func IssueTokenPair(user *domain.User) (*TokenPair, error) {
	accessToken, err := IssueAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := IssueRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// IssueToken creates a new JWT for the given user (legacy, uses access token)
// Deprecated: Use IssueAccessToken or IssueTokenPair instead
func IssueToken(user *domain.User) (string, error) {
	return IssueAccessToken(user)
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

// ValidateAccessToken validates a token and ensures it's an access token
func ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeAccess {
		return nil, errors.New("invalid token type: expected access token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a token and ensures it's a refresh token
func ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, errors.New("invalid token type: expected refresh token")
	}

	return claims, nil
}
