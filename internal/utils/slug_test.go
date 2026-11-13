package utils

import (
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
}

