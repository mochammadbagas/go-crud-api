package mailer

import "context"

// Mailer defines the contract for sending transactional emails across the application.
type Mailer interface {
	SendResetPasswordEmail(ctx context.Context, toEmail, resetLink string) error
	SendVerificationEmail(ctx context.Context, toEmail, verifyLink string) error
	SendEmailChangeEmail(ctx context.Context, toNewEmail, confirmLink string) error
}
