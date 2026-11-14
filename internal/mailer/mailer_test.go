package mailer_test

import (
	"context"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/mailer"
)

func TestLogMailer(t *testing.T) {
	m := mailer.NewLogMailer()
	ctx := context.Background()

	if err := m.SendResetPasswordEmail(ctx, "user@example.com", "http://localhost:3000/reset?token=123"); err != nil {
		t.Errorf("expected no error from LogMailer reset, got: %v", err)
	}

	if err := m.SendVerificationEmail(ctx, "user@example.com", "http://localhost:3000/verify?token=456"); err != nil {
		t.Errorf("expected no error from LogMailer verify, got: %v", err)
	}

	if err := m.SendEmailChangeEmail(ctx, "new@example.com", "http://localhost:3000/confirm?token=789"); err != nil {
		t.Errorf("expected no error from LogMailer email change, got: %v", err)
	}
}

func TestMailerFactory(t *testing.T) {
	// Test default fallback to LogMailer
	cfgDev := &config.Config{
		MailerDriver: "log",
	}
	mDev := mailer.New(cfgDev)
	if mDev == nil {
		t.Fatal("expected mailer to be non-nil")
	}

	// Test resend without API key fallbacks to LogMailer
	cfgNoKey := &config.Config{
		MailerDriver: "resend",
		ResendAPIKey: "",
	}
	mNoKey := mailer.New(cfgNoKey)
	if mNoKey == nil {
		t.Fatal("expected fallback mailer to be non-nil")
	}

	// Test resend with API key
	cfgResend := &config.Config{
		MailerDriver: "resend",
		ResendAPIKey: "re_test_key_12345",
		MailFrom:     "Test <test@example.com>",
	}
	mResend := mailer.New(cfgResend)
	if mResend == nil {
		t.Fatal("expected ResendMailer to be non-nil")
	}
}
