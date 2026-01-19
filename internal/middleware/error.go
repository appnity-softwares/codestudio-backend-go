package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/pkg/errors"
	"github.com/pushp314/devconnect-backend/pkg/logger"
)

// ErrorHandlerMiddleware handles errors and panics
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log panic stack trace
				stack := string(debug.Stack())
				logger.Error().
					Str("panic", fmt.Sprintf("%v", r)).
					Str("stack", stack).
					Msg("Panic recovered")

				// Return 500 error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal Server Error",
					"message": "An unexpected error occurred",
				})
				c.Abort()
			}
		}()

		c.Next()

		// Handle errors attached to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Check if it's our custom AppError
			if appErr, ok := err.(*errors.AppError); ok {
				c.JSON(appErr.Code, gin.H{
					"error": appErr.Message,
				})
				return
			}

			// Handle other errors (default to 500 if not specified)
			// In a real app, you might want to map specific errors (db, val, etc)
			logger.Error().Err(err).Msg("Unhandled request error")

			// Don't expose internal errors to client in production unless safe
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal Server Error",
			})
		}
	}
}
