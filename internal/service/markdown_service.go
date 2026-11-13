package service

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// MarkdownRenderer defines the contract for rendering Markdown to safe HTML.
// Using an interface enables swapping implementations and simplifies testing.
type MarkdownRenderer interface {
	RenderToHTML(source string) (string, error)
}

// MarkdownService handles safe markdown conversion to HTML.
type MarkdownService struct {
	md goldmark.Markdown
}

// NewMarkdownService creates a configured, XSS-safe Markdown renderer.
func NewMarkdownService() *MarkdownService {
	// NFR-005: Markdown rendered safely to prevent XSS.
	// By default, goldmark does not enable html.WithUnsafe(), meaning raw HTML tags are safely escaped or suppressed.
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &MarkdownService{md: md}
}

// RenderToHTML renders a markdown string into safe HTML.
func (s *MarkdownService) RenderToHTML(source string) (string, error) {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

