package domain

import "time"

type OAuthProvider string

const (
	OAuthProviderGoogle  OAuthProvider = "google"
	OAuthProviderDiscord OAuthProvider = "discord"
)

type User struct {
	ID            uint64
	Email         string
	DisplayName   string
	AvatarURL     *string
	OAuthProvider *OAuthProvider // nil for password-only users
	OAuthID       *string        // nil for password-only users
	Username      *string        // nil for OAuth-only users
	PasswordHash  *string        // nil for OAuth-only users
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
