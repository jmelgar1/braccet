package service

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridEmailSender sends emails via SendGrid API
type SendGridEmailSender struct {
	apiKey    string
	fromEmail string
	fromName  string
}

// NewSendGridEmailSender creates a new SendGrid email sender
func NewSendGridEmailSender(cfg EmailConfig) *SendGridEmailSender {
	return &SendGridEmailSender{
		apiKey:    cfg.APIKey,
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
	}
}

func (s *SendGridEmailSender) SendVerificationEmail(ctx context.Context, to, verificationURL string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	toEmail := mail.NewEmail("", to)
	subject := "Verify your Braccet account"

	plainTextContent := fmt.Sprintf(`Welcome to Braccet!

Click the link below to verify your email address:
%s

This link expires in 1 hour.

If you didn't create this account, you can ignore this email.
`, verificationURL)

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { margin-top: 30px; font-size: 14px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to Braccet!</h1>
        <p>Click the button below to verify your email address:</p>
        <a href="%s" class="button">Verify Email</a>
        <p>Or copy and paste this link into your browser:</p>
        <p><a href="%s">%s</a></p>
        <p class="footer">This link expires in 1 hour.<br>If you didn't create this account, you can ignore this email.</p>
    </div>
</body>
</html>
`, verificationURL, verificationURL, verificationURL)

	message := mail.NewSingleEmail(from, subject, toEmail, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(s.apiKey)

	response, err := client.SendWithContext(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error: status %d, body: %s", response.StatusCode, response.Body)
	}

	return nil
}

// ConsoleEmailSender logs emails to console (for development)
type ConsoleEmailSender struct{}

// NewConsoleEmailSender creates an email sender that logs to console
func NewConsoleEmailSender() *ConsoleEmailSender {
	return &ConsoleEmailSender{}
}

func (s *ConsoleEmailSender) SendVerificationEmail(ctx context.Context, to, verificationURL string) error {
	fmt.Printf("\n========== VERIFICATION EMAIL ==========\n")
	fmt.Printf("To: %s\n", to)
	fmt.Printf("Subject: Verify your Braccet account\n")
	fmt.Printf("Verification URL: %s\n", verificationURL)
	fmt.Printf("=========================================\n\n")
	return nil
}
