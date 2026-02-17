package service

import "context"

// EmailSender abstracts email delivery to support multiple providers
type EmailSender interface {
	// SendVerificationEmail sends a verification link to the given email address
	SendVerificationEmail(ctx context.Context, to, verificationURL string) error
}

// EmailConfig holds configuration for the email service
type EmailConfig struct {
	Provider     string // "sendgrid", "ses", "console" (for dev)
	APIKey       string // For SendGrid/SES
	FromEmail    string
	FromName     string
}
