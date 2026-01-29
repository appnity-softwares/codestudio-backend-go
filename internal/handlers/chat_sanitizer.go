package handlers

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Message length limits
const (
	MaxMessageLength = 8000  // Characters for regular text (increased)
	MaxCodeLength    = 20000 // For code snippets (increased)
	MinMessageLength = 1     // Minimum message length
)

// Dangerous patterns for XSS prevention
var (
	scriptTagRegex = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	onEventRegex   = regexp.MustCompile(`(?i)\s+on\w+\s*=`)
)

// SanitizeMessageContent cleans and validates message content
// Returns sanitized content or error if validation fails
func SanitizeMessageContent(content string, msgType string) (string, error) {
	// 1. Check minimum length
	if strings.TrimSpace(content) == "" {
		return "", errors.New("message cannot be empty")
	}

	// 2. Check maximum length based on type
	maxLen := MaxMessageLength
	if msgType == "code" {
		maxLen = MaxCodeLength
	}
	if utf8.RuneCountInString(content) > maxLen {
		return "", errors.New("message exceeds maximum length")
	}

	// 3. For code messages, only do minimal sanitization (preserve formatting)
	if msgType == "code" {
		// Remove script tags even in code (they shouldn't execute but better safe)
		content = scriptTagRegex.ReplaceAllString(content, "[script removed]")
		return content, nil
	}

	// 4. For text messages, full sanitization
	// Remove script tags
	content = scriptTagRegex.ReplaceAllString(content, "")

	// Remove inline event handlers (onclick, onload, etc.)
	content = onEventRegex.ReplaceAllString(content, " ")

	// Escape HTML entities to prevent XSS
	content = html.EscapeString(content)

	// 5. Trim whitespace
	content = strings.TrimSpace(content)

	if content == "" {
		return "", errors.New("message cannot be empty after sanitization")
	}

	return content, nil
}

// ValidateMessageType checks if the message type is valid
func ValidateMessageType(msgType string) bool {
	validTypes := map[string]bool{
		"text":   true,
		"code":   true,
		"image":  true,
		"system": true,
	}
	return validTypes[msgType]
}

// SanitizeCodeMetadata validates and sanitizes code snippet metadata
// Expected format: {"language": "go", "filename": "main.go"}
func SanitizeCodeMetadata(metadata string) string {
	// For now, just ensure it's valid JSON-ish or empty
	// Full validation can be added later
	if metadata == "" {
		return "{}"
	}
	// Basic check - should start with { and end with }
	metadata = strings.TrimSpace(metadata)
	if !strings.HasPrefix(metadata, "{") || !strings.HasSuffix(metadata, "}") {
		return "{}"
	}
	return metadata
}
