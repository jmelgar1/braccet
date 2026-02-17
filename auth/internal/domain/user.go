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
	OAuthProvider OAuthProvider
	OAuthID       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
