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
	printMailBox("RESET PASSWORD", toEmail, "Instruksi Pemulihan Kata Sandi Akun", resetLink, "15 Menit")
	return nil
}

func (m *logMailer) SendVerificationEmail(ctx context.Context, toEmail, verifyLink string) error {
	printMailBox("VERIFIKASI EMAIL AKUN BARU", toEmail, "Verifikasi Alamat Email Anda", verifyLink, "24 Jam")
	return nil
}

func (m *logMailer) SendEmailChangeEmail(ctx context.Context, toNewEmail, confirmLink string) error {
	printMailBox("KONFIRMASI PERUBAHAN EMAIL", toNewEmail, "Konfirmasi Alamat Email Baru Anda", confirmLink, "1 Jam")
	return nil
}

func printMailBox(tag, to, subject, link, ttl string) {
	width := 76
	divider := "┌" + strings.Repeat("─", width-2) + "┐"
	bottomDivider := "└" + strings.Repeat("─", width-2) + "┘"
	title := fmt.Sprintf("│ 📧 [MAILER DEV / LOG] - %s", tag)
	titlePad := width - len([]rune(title)) - 1
	if titlePad < 0 {
		titlePad = 0
	}

	fmt.Println()
	fmt.Println(divider)
	fmt.Println(title + strings.Repeat(" ", titlePad) + "│")
	fmt.Println("├" + strings.Repeat("─", width-2) + "┤")
	fmt.Printf("│ Kepada   : %-64s│\n", truncateString(to, 64))
	fmt.Printf("│ Subjek   : %-64s│\n", truncateString(subject, 64))
	fmt.Printf("│ Berlaku  : %-64s│\n", truncateString(ttl, 64))
	fmt.Println("├" + strings.Repeat("─", width-2) + "┤")
	fmt.Println("│ Tautan Aksi:")
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
