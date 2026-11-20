package utils

import (
	"strings"
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic title",
			input:    "Building REST APIs with Go",
			expected: "building-rest-apis-with-go",
		},
		{
			name:     "With punctuation and special characters",
			input:    "Go 1.25 & MySQL 8.0: The Ultimate Guide!",
			expected: "go-125-mysql-80-the-ultimate-guide",
		},
		{
			name:     "With multiple spaces and hyphens",
			input:    "   Rust   vs   Go --- File Hashing   ",
			expected: "rust-vs-go-file-hashing",
		},
		{
			name:     "Uppercase and lowercase mixed",
			input:    "My First JOURNAL Entry",
			expected: "my-first-journal-entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			if got != tt.expected {
				t.Errorf("GenerateSlug(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}

	t.Run("Non-Latin title fallback to post-ULID", func(t *testing.T) {
		got := GenerateSlug("こんにちは 世界")
		if len(got) < 10 || !strings.HasPrefix(got, "post-") {
			t.Errorf("expected non-Latin slug to start with 'post-', got %q", got)
		}
	})

	t.Run("Emoji-only title fallback to post-ULID", func(t *testing.T) {
		got := GenerateSlug("🚀🔥🎉 (???)")
		if len(got) < 10 || !strings.HasPrefix(got, "post-") {
			t.Errorf("expected emoji-only slug to start with 'post-', got %q", got)
		}
	})

	t.Run("Overly long title capped at 255 chars without trailing hyphen", func(t *testing.T) {
		veryLongTitle := strings.Repeat("golang-crud-api-testing-", 20)
		got := GenerateSlug(veryLongTitle)
		if len(got) > 255 {
			t.Errorf("expected slug length <= 255, got %d", len(got))
		}
		if strings.HasSuffix(got, "-") {
			t.Errorf("expected slug not to end with hyphen, got %q", got)
		}
	})
}

