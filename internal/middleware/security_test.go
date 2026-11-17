package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(rec, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.expected {
			t.Errorf("Header %q = %q; want %q", tt.header, got, tt.expected)
		}
	}
}
