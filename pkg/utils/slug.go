package utils

import (
	"regexp"
	"strings"
)

// GenerateSlug creates a URL-friendly slug from a string
func GenerateSlug(input string) string {
	// Convert to lowercase
	slug := strings.ToLower(input)
	// Remove non-alphanumeric characters (except spaces)
	reg, _ := regexp.Compile("[^a-z0-9 ]+")
	slug = reg.ReplaceAllString(slug, "")
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
