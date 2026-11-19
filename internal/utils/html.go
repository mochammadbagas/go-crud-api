package utils

import (
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

var (
	htmlPolicy     *bluemonday.Policy
	htmlPolicyOnce sync.Once
)

// getHTMLPolicy initializes and caches the Bluemonday sanitizer policy.
func getHTMLPolicy() *bluemonday.Policy {
	htmlPolicyOnce.Do(func() {
		p := bluemonday.UGCPolicy()

		// Allow images with custom attributes such as data-media-id, responsive attrs, and styling classes
		p.AllowAttrs("src", "alt", "title", "width", "height", "loading", "class", "data-media-id").OnElements("img")

		// Ensure iframe embedding is only allowed from safe domains (e.g. YouTube, Vimeo) if needed,
		// or disallow raw iframes entirely by default for strict security.
		p.RequireParseableURLs(true)
		p.AllowRelativeURLs(true)

		htmlPolicy = p
	})
	return htmlPolicy
}

// SanitizeHTML strips malicious tags, scripts, and insecure URL schemes from HTML content.
func SanitizeHTML(rawHTML string) string {
	if strings.TrimSpace(rawHTML) == "" {
		return ""
	}
	return getHTMLPolicy().Sanitize(rawHTML)
}

// StripHTMLTags removes all HTML tags and leaves clean plain text.
func StripHTMLTags(htmlContent string) string {
	if strings.TrimSpace(htmlContent) == "" {
		return ""
	}
	return bluemonday.StrictPolicy().Sanitize(htmlContent)
}

// ExtractMediaIDsFromHTML scans the HTML content for <img> tags and collects all unique data-media-id values.
func ExtractMediaIDsFromHTML(htmlContent string) []string {
	if strings.TrimSpace(htmlContent) == "" {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var mediaIDs []string

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "data-media-id" {
					val := strings.TrimSpace(attr.Val)
					if val != "" && !seen[val] {
						seen[val] = true
						mediaIDs = append(mediaIDs, val)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return mediaIDs
}
