package domain

import "time"

// PendingRegistration represents a signup awaiting email verification.
// Once verified, the data is moved to the users table and this record is deleted.
type PendingRegistration struct {
	ID                uint64
	Email             string
	Username          string
	DisplayName       string
	PasswordHash      string
	VerificationToken string
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// IsExpired returns true if the verification token has expired
func (p *PendingRegistration) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
