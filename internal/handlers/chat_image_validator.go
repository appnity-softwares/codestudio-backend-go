package handlers

import (
	"errors"
	"net/url"
	"strings"
)

// Allowlisted image hosts for security
// Expand this list as needed, but keep it curated
var allowedImageHosts = []string{
	// Popular developer-friendly image hosts
	"images.unsplash.com",
	"source.unsplash.com",
	"cdn.jsdelivr.net",
	"raw.githubusercontent.com",
	"user-images.githubusercontent.com",
	"avatars.githubusercontent.com",
	"i.imgur.com",
	"imgur.com",
	// Cloud storage (public buckets)
	"storage.googleapis.com",
	"s3.amazonaws.com",
	// CDNs
	"res.cloudinary.com",
	"imagedelivery.net",
	// Placeholder/dev tools
	"via.placeholder.com",
	"picsum.photos",
	"placekitten.com",
	"placehold.co",
}

// Allowed image file extensions
var allowedImageExtensions = []string{
	".png",
	".jpg",
	".jpeg",
	".webp",
	".gif",
	".svg", // SVG is safe when served from trusted hosts with proper headers
}

// Maximum URL length to prevent abuse
const maxImageURLLength = 2048

// ValidateImageURL validates that a URL is a safe, allowed image URL
func ValidateImageURL(imageURL string) error {
	// Check URL length
	if len(imageURL) > maxImageURLLength {
		return errors.New("image URL too long (max 2048 characters)")
	}

	// Trim and validate
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return errors.New("image URL cannot be empty")
	}

	// Parse URL
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return errors.New("invalid image URL format")
	}

	// SECURITY: Only HTTPS allowed
	if parsed.Scheme != "https" {
		return errors.New("only HTTPS image URLs are allowed")
	}

	// SECURITY: Block dangerous URL schemes
	lowerURL := strings.ToLower(imageURL)
	if strings.HasPrefix(lowerURL, "javascript:") ||
		strings.HasPrefix(lowerURL, "data:") ||
		strings.HasPrefix(lowerURL, "vbscript:") ||
		strings.Contains(lowerURL, "<script") ||
		strings.Contains(lowerURL, "onerror=") {
		return errors.New("unsafe image URL detected")
	}

	// Check host allowlist
	hostAllowed := false
	for _, allowedHost := range allowedImageHosts {
		if strings.HasSuffix(parsed.Host, allowedHost) || parsed.Host == allowedHost {
			hostAllowed = true
			break
		}
	}
	if !hostAllowed {
		return errors.New("image host not in allowlist. Supported: Unsplash, GitHub, Imgur, Cloudinary")
	}

	// Check file extension (optional but recommended)
	// Some CDNs serve images without extensions, so we're lenient here
	lowerPath := strings.ToLower(parsed.Path)
	hasValidExtension := false
	for _, ext := range allowedImageExtensions {
		if strings.HasSuffix(lowerPath, ext) {
			hasValidExtension = true
			break
		}
	}

	// For CDNs that don't use extensions in path, check query params or just allow
	if !hasValidExtension {
		// Check if it's a known CDN that doesn't require extensions
		noExtensionHosts := []string{
			"picsum.photos",
			"source.unsplash.com",
			"via.placeholder.com",
			"placehold.co",
			"imagedelivery.net",
			"res.cloudinary.com",
		}
		for _, host := range noExtensionHosts {
			if strings.Contains(parsed.Host, host) {
				hasValidExtension = true
				break
			}
		}
	}

	if !hasValidExtension {
		return errors.New("URL must point to an image file (.png, .jpg, .jpeg, .webp, .gif, .svg)")
	}

	return nil
}

// IsAllowedImageHost checks if a host is in the allowlist
func IsAllowedImageHost(host string) bool {
	for _, allowed := range allowedImageHosts {
		if strings.HasSuffix(host, allowed) || host == allowed {
			return true
		}
	}
	return false
}
