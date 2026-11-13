package service

import (
	"strings"
	"testing"
)

func TestMarkdownService_RenderToHTML(t *testing.T) {
	svc := NewMarkdownService()

	t.Run("Renders headers and paragraphs", func(t *testing.T) {
		input := "# Judul Blog\n\nIni adalah konten paragraf."
		html, err := svc.RenderToHTML(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, "<h1") || !strings.Contains(html, "Judul Blog") {
			t.Errorf("expected h1 tag with title, got: %s", html)
		}
		if !strings.Contains(html, "<p>Ini adalah konten paragraf.</p>") {
			t.Errorf("expected paragraph tag, got: %s", html)
		}
	})

	t.Run("Prevents raw script XSS injection (NFR-005)", func(t *testing.T) {
		input := "Contoh <script>alert('xss')</script> teks aman."
		html, err := svc.RenderToHTML(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// In safe goldmark mode, raw HTML tags like <script> are stripped or escaped
		if strings.Contains(html, "<script>alert('xss')</script>") {
			t.Errorf("expected script tag to be escaped or eliminated, got: %s", html)
		}
	})
}

