package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9\s-]`)
	duplicateHyphenRegex = regexp.MustCompile(`-+`)
)

// GenerateSlug transforms a title into a URL-friendly, lowercase, hyphen-separated slug.
func GenerateSlug(title string) string {
	// 1. Lowercase
	s := strings.ToLower(title)

	// 2. Remove accents or map runes
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// 3. Remove non-alphanumeric (except hyphen and whitespace)
	s = nonAlphanumericRegex.ReplaceAllString(s, "")

	// 4. Replace whitespace with hyphens
	s = strings.Join(strings.Fields(s), "-")

	// 5. Deduplicate hyphens
	s = duplicateHyphenRegex.ReplaceAllString(s, "-")

	// 6. Trim leading and trailing hyphens
	s = strings.Trim(s, "-")

	// 7. Enforce maximum length limit of 255 characters (MySQL VARCHAR(255) safe)
	if len(s) > 255 {
		s = s[:255]
		s = strings.TrimRight(s, "-")
	}

	// 8. Fallback for non-Latin scripts or pure emoji titles
	if s == "" {
		return fmt.Sprintf("post-%s", strings.ToLower(NewULID()))
	}

	return s
}

