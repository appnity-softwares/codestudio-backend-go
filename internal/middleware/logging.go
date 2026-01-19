package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/pkg/logger"
)

// LoggingMiddleware logs all incoming requests with timing
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Get user ID if authenticated
		userId, _ := c.Get("userId")
		userIdStr := ""
		if userId != nil {
			userIdStr = userId.(string)
		}

		// Build log event
		event := logger.Log.Info()
		if status >= 400 {
			event = logger.Log.Warn()
		}
		if status >= 500 {
			event = logger.Log.Error()
		}

		event.
			Str("method", method).
			Str("path", path).
			Str("query", rawQuery).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", clientIP).
			Str("user_agent", userAgent).
			Str("user_id", userIdStr).
			Int("body_size", c.Writer.Size()).
			Msg("request")
	}
}
