package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type resendMailer struct {
	apiKey     string
	fromAddr   string
	httpClient *http.Client
}

// NewResendMailer creates a production mailer communicating via Resend REST API.
func NewResendMailer(apiKey, fromAddr string) Mailer {
	return &resendMailer{
		apiKey:   apiKey,
		fromAddr: fromAddr,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (m *resendMailer) send(ctx context.Context, toEmail, subject, htmlContent string) error {
	if m.apiKey == "" {
		return errors.New("resend API key belum dikonfigurasi di environment RESEND_API_KEY")
	}

	payload := resendPayload{
		From:    m.fromAddr,
		To:      []string{toEmail},
		Subject: subject,
		HTML:    htmlContent,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal menyusun payload email: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("gagal membuat HTTP request resend: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengirim email via Resend API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API mengembalikan status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (m *resendMailer) SendResetPasswordEmail(ctx context.Context, toEmail, resetLink string) error {
	subject := "Instruksi Pemulihan Kata Sandi Akun"
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f6f9fc; padding: 40px 20px;">
  <div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 8px; padding: 32px; box-shadow: 0 2px 4px rgba(0,0,0,0.06);">
    <h2 style="color: #1a1a1a; margin-top: 0;">Pemulihan Kata Sandi</h2>
    <p style="color: #4a5568; line-height: 1.6;">Kami menerima permintaan untuk mereset kata sandi akun Anda. Klik tombol di bawah ini untuk melanjutkan:</p>
    <div style="margin: 28px 0;">
      <a href="%s" style="background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 600; display: inline-block;">Atur Ulang Kata Sandi</a>
    </div>
    <p style="color: #718096; font-size: 14px; line-height: 1.5;">Tautan ini hanya berlaku selama <strong>15 menit</strong>. Jika Anda tidak merasa meminta reset kata sandi, abaikan email ini dengan aman.</p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
    <p style="color: #a0aec0; font-size: 12px; margin-bottom: 0;">Jika tombol di atas tidak berfungsi, salin tautan berikut ke peramban Anda:<br/><a href="%s" style="color: #2563eb; word-break: break-all;">%s</a></p>
  </div>
</body>
</html>
`, resetLink, resetLink, resetLink)

	return m.send(ctx, toEmail, subject, html)
}

func (m *resendMailer) SendVerificationEmail(ctx context.Context, toEmail, verifyLink string) error {
	subject := "Verifikasi Alamat Email Anda"
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f6f9fc; padding: 40px 20px;">
  <div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 8px; padding: 32px; box-shadow: 0 2px 4px rgba(0,0,0,0.06);">
    <h2 style="color: #1a1a1a; margin-top: 0;">Selamat Datang!</h2>
    <p style="color: #4a5568; line-height: 1.6;">Terima kasih telah mendaftar. Silakan klik tombol di bawah ini untuk memverifikasi alamat email akun Anda:</p>
    <div style="margin: 28px 0;">
      <a href="%s" style="background-color: #10b981; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 600; display: inline-block;">Verifikasi Email Saya</a>
    </div>
    <p style="color: #718096; font-size: 14px; line-height: 1.5;">Tautan ini berlaku selama <strong>24 jam</strong>.</p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
    <p style="color: #a0aec0; font-size: 12px; margin-bottom: 0;">Atau salin tautan berikut:<br/><a href="%s" style="color: #10b981; word-break: break-all;">%s</a></p>
  </div>
</body>
</html>
`, verifyLink, verifyLink, verifyLink)

	return m.send(ctx, toEmail, subject, html)
}

func (m *resendMailer) SendEmailChangeEmail(ctx context.Context, toNewEmail, confirmLink string) error {
	subject := "Konfirmasi Perubahan Alamat Email"
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f6f9fc; padding: 40px 20px;">
  <div style="max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 8px; padding: 32px; box-shadow: 0 2px 4px rgba(0,0,0,0.06);">
    <h2 style="color: #1a1a1a; margin-top: 0;">Konfirmasi Alamat Email Baru</h2>
    <p style="color: #4a5568; line-height: 1.6;">Kami menerima permintaan untuk mengubah alamat email akun Anda ke alamat ini (%s). Klik tombol di bawah untuk menyetujui:</p>
    <div style="margin: 28px 0;">
      <a href="%s" style="background-color: #6366f1; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 600; display: inline-block;">Konfirmasi Email Baru</a>
    </div>
    <p style="color: #718096; font-size: 14px; line-height: 1.5;">Tautan ini hanya berlaku selama <strong>1 jam</strong>. Jika Anda tidak merasa melakukan permintaan ini, abaikan email ini.</p>
    <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 24px 0;" />
    <p style="color: #a0aec0; font-size: 12px; margin-bottom: 0;">Atau salin tautan berikut:<br/><a href="%s" style="color: #6366f1; word-break: break-all;">%s</a></p>
  </div>
</body>
</html>
`, toNewEmail, confirmLink, confirmLink, confirmLink)

	return m.send(ctx, toNewEmail, subject, html)
}
