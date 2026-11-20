package middleware

import (
	"net/http"
)

// SecurityHeaders returns an HTTP middleware that injects recommended OWASP security headers.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking by forbidding embedding in frames/iframes
			w.Header().Set("X-Frame-Options", "DENY")

			// Control referrer information sent in HTTP requests
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// XSS protection filter for legacy browsers
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Content Security Policy default safe fallback for JSON APIs and static media assets
			w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; media-src 'self'; frame-ancestors 'none'")

			// HTTP Strict Transport Security (HSTS) when served over HTTPS (W3-SEC-03)
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
