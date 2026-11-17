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

			// Content Security Policy default safe fallback for JSON APIs
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

			next.ServeHTTP(w, r)
		})
	}
}
