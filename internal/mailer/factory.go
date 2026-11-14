package mailer

import (
	"log/slog"

	"github.com/mochammadbagas/go-crud-api/internal/config"
)

// New initializes the appropriate Mailer adapter based on configuration.
func New(cfg *config.Config) Mailer {
	if cfg.MailerDriver == "resend" {
		if cfg.ResendAPIKey == "" {
			slog.Warn("Peringatan: MAILER_DRIVER disetel 'resend' tetapi RESEND_API_KEY kosong. Fallback otomatis ke LogMailer.")
			return NewLogMailer()
		}
		slog.Info("Menginisialisasi ResendMailer (Mode Production)", slog.String("from", cfg.MailFrom))
		return NewResendMailer(cfg.ResendAPIKey, cfg.MailFrom)
	}

	slog.Info("Menginisialisasi LogMailer (Mode Development / Outbox Terminal)")
	return NewLogMailer()
}
