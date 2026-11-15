package mailer

import (
	"context"
	"fmt"
	"strings"
)

type logMailer struct{}

// NewLogMailer creates a zero-dependency mailer that logs outgoing emails to stdout.
func NewLogMailer() Mailer {
	return &logMailer{}
}

func (m *logMailer) SendResetPasswordEmail(ctx context.Context, toEmail, resetLink string) error {
	printMailBox("PASSWORD RESET", toEmail, "Account Password Reset Instructions", resetLink, "15 Minutes")
	return nil
}

func (m *logMailer) SendVerificationEmail(ctx context.Context, toEmail, verifyLink string) error {
	printMailBox("NEW ACCOUNT EMAIL VERIFICATION", toEmail, "Verify Your Email Address", verifyLink, "24 Hours")
	return nil
}

func (m *logMailer) SendEmailChangeEmail(ctx context.Context, toNewEmail, confirmLink string) error {
	printMailBox("EMAIL CHANGE CONFIRMATION", toNewEmail, "Confirm Your New Email Address", confirmLink, "1 Hour")
	return nil
}

func printMailBox(tag, to, subject, link, ttl string) {
	width := 76
	divider := "┌" + strings.Repeat("─", width-2) + "┐"
	bottomDivider := "└" + strings.Repeat("─", width-2) + "┘"
	title := fmt.Sprintf("│ 📧 [DEV MAILER / CONSOLE LOG] - %s", tag)
	titlePad := width - len([]rune(title)) - 1
	if titlePad < 0 {
		titlePad = 0
	}

	fmt.Println()
	fmt.Println(divider)
	fmt.Println(title + strings.Repeat(" ", titlePad) + "│")
	fmt.Println("├" + strings.Repeat("─", width-2) + "┤")
	fmt.Printf("│ To       : %-64s│\n", truncateString(to, 64))
	fmt.Printf("│ Subject  : %-64s│\n", truncateString(subject, 64))
	fmt.Printf("│ Expires  : %-64s│\n", truncateString(ttl, 64))
	fmt.Println("├" + strings.Repeat("─", width-2) + "┤")
	fmt.Println("│ Action Link:")
	fmt.Printf("│ %s\n", link)
	fmt.Println(bottomDivider)
	fmt.Println()
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
