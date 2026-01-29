package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds various security headers to the response
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Strict Transport Security (HSTS) - 1 year in seconds, include subdomains
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Clickjacking protection
		c.Header("X-Frame-Options", "DENY")

		// XSS protection (for older browsers, modern ones use CSP)
		c.Header("X-XSS-Protection", "1; mode=block")

		// MIME type sniffing protection
		c.Header("X-Content-Type-Options", "nosniff")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (CSP)
		// Relaxed for now to allow external images/scripts common in dev
		// In production, this should be stricter
		c.Header("Content-Security-Policy", "default-src 'self' https:; script-src 'self' 'unsafe-inline' 'unsafe-eval' https:; style-src 'self' 'unsafe-inline' https:; img-src 'self' data: https:; font-src 'self' https: data:; worker-src 'self' blob:; connect-src 'self' https: wss:;")

		// Permissions Policy
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
