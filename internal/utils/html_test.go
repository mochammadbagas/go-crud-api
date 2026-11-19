package utils

import (
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "Strips script tags",
			input:    `<p>Hello <script>alert('xss')</script>World</p>`,
			contains: `<p>Hello World</p>`,
			excludes: `<script>`,
		},
		{
			name:     "Strips onload and onclick handlers",
			input:    `<img src="test.jpg" onerror="alert(1)" onclick="steal()" data-media-id="01HXYZ">`,
			contains: `data-media-id="01HXYZ"`,
			excludes: `onerror`,
		},
		{
			name:     "Preserves safe typography and media attributes",
			input:    `<h1>Title</h1><p>Paragraph with <a href="https://example.com">link</a> and <img src="/img.png" data-media-id="01ABC"></p>`,
			contains: `data-media-id="01ABC"`,
			excludes: ``,
		},
		{
			name:     "Rejects javascript: pseudo-protocol",
			input:    `<a href="javascript:alert(1)">Click</a>`,
			contains: `Click`,
			excludes: `javascript:`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SanitizeHTML(tt.input)
			if tt.contains != "" && !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got %q", tt.contains, output)
			}
			if tt.excludes != "" && strings.Contains(output, tt.excludes) {
				t.Errorf("expected output NOT to contain %q, got %q", tt.excludes, output)
			}
		})
	}
}

func TestExtractMediaIDsFromHTML(t *testing.T) {
	htmlContent := `
		<div>
			<p>Intro text</p>
			<img src="https://cdn.example.com/1.jpg" data-media-id="MEDIA_01" alt="Image 1">
			<p>Middle text</p>
			<img src="/local/2.jpg" data-media-id="MEDIA_02" />
			<img src="/local/duplicate.jpg" data-media-id="MEDIA_01" />
			<img src="/local/no-id.jpg" alt="No data id" />
		</div>
	`

	ids := ExtractMediaIDsFromHTML(htmlContent)
	if len(ids) != 2 {
		t.Fatalf("expected 2 unique media IDs, got %d: %v", len(ids), ids)
	}

	if ids[0] != "MEDIA_01" || ids[1] != "MEDIA_02" {
		t.Errorf("unexpected extracted IDs: %v", ids)
	}
}
