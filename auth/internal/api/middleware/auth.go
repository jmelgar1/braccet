package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/braccet/auth/internal/service"
)

type contextKey string

const (
	UserIDKey      contextKey = "user_id"
	UserEmailKey   contextKey = "user_email"
	UserNameKey    contextKey = "user_name"
)

// Auth middleware validates JWT access tokens and adds user info to context
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"authorization header required"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		claims, err := service.ValidateAccessToken(tokenString)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, UserNameKey, claims.DisplayName)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) (uint64, bool) {
	userID, ok := ctx.Value(UserIDKey).(uint64)
	return userID, ok
}

// GetUserEmail extracts user email from context
func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

// GetUserName extracts user display name from context
func GetUserName(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(UserNameKey).(string)
	return name, ok
}
