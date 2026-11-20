package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Logger returns an HTTP middleware that logs incoming requests with structured slog.
// It includes noise reduction (skipping /health & /ready on 200 OK) and sensitive header redaction in production.
func Logger(cfg *config.Config) func(next http.Handler) http.Handler {
	isProd := cfg != nil && cfg.AppEnv == "production"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Noise reduction: Skip logging health & readiness check polling when status is OK
			isHealthProbe := path == "/health" || path == "/ready" || path == "/live"

			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				status := ww.Status()
				if status == 0 {
					status = http.StatusOK
				}

				// If it's a health probe and response was successful (< 400), don't pollute logs
				if isHealthProbe && status < 400 {
					return
				}

				duration := time.Since(start)
				reqID := chimiddleware.GetReqID(r.Context())

				attrs := []slog.Attr{
					slog.String("request_id", reqID),
					slog.String("method", r.Method),
					slog.String("path", path),
					slog.Int("status", status),
					slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
					slog.Int("bytes", ww.BytesWritten()),
					slog.String("ip", r.RemoteAddr),
				}

				if ua := r.UserAgent(); ua != "" {
					attrs = append(attrs, slog.String("user_agent", ua))
				}

				// Sanitized headers (Sensitive data protection - OWASP CWE-532)
				if authHeader := r.Header.Get("Authorization"); authHeader != "" {
					if isProd {
						attrs = append(attrs, slog.String("auth", maskAuthHeader(authHeader)))
					} else {
						attrs = append(attrs, slog.String("auth", authHeader))
					}
				}

				// Intelligent log levels based on HTTP status
				ctx := r.Context()
				switch {
				case status >= 500:
					slog.LogAttrs(ctx, slog.LevelError, "HTTP request completed with server error", attrs...)
				case status >= 400:
					slog.LogAttrs(ctx, slog.LevelWarn, "HTTP request completed with client error", attrs...)
				default:
					slog.LogAttrs(ctx, slog.LevelInfo, "HTTP request completed successfully", attrs...)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// maskAuthHeader masks a Bearer token or credentials for production safety (CWE-532).
func maskAuthHeader(auth string) string {
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		token := parts[1]
		if len(token) > 10 {
			return "Bearer " + token[:6] + "..." + token[len(token)-4:]
		}
		return "Bearer [REDACTED]"
	}
	return "[REDACTED]"
}
