package utils

import (
	"html"
	"regexp"
	"strings"
)

// sanitize.go - Input sanitization utilities for security

// EscapeSQLWildcards escapes SQL LIKE/ILIKE wildcard characters to prevent injection
// This is used when user input is used in LIKE/ILIKE queries
func EscapeSQLWildcards(input string) string {
	// Escape backslash first (as it's the escape character)
	input = strings.ReplaceAll(input, "\\", "\\\\")
	// Escape SQL wildcards
	input = strings.ReplaceAll(input, "%", "\\%")
	input = strings.ReplaceAll(input, "_", "\\_")
	return input
}

// SanitizeSearchQuery prepares a search string for safe ILIKE usage
// Returns the sanitized term wrapped with % for partial matching
func SanitizeSearchQuery(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)
	// Limit length to prevent DoS
	if len(input) > 100 {
		input = input[:100]
	}
	// Escape wildcards
	input = EscapeSQLWildcards(input)
	return "%" + input + "%"
}

// SanitizeHTML escapes HTML entities to prevent XSS
// Use this for any user-generated content that will be displayed
func SanitizeHTML(input string) string {
	return html.EscapeString(input)
}

// StripHTML removes all HTML tags from a string
// More aggressive than SanitizeHTML - removes tags entirely
func StripHTML(input string) string {
	// Simple regex to strip HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// ValidateUsername checks if username contains only allowed characters
// Returns true if valid
func ValidateUsername(username string) bool {
	// Allow alphanumeric, underscores, hyphens. 3-30 characters
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)
	return re.MatchString(username)
}

// TruncateString safely truncates a string to max length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
