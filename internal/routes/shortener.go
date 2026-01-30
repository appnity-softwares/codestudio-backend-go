package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
)

// RegisterShortenerRoutes registers the root redirection route
func RegisterShortenerRoutes(r *gin.Engine) {
	r.GET("/s/:code", handlers.RedirectShortLink)
}

// RegisterShortenerAPIRoutes registers the creation endpoint
func RegisterShortenerAPIRoutes(r gin.IRouter) {
	r.POST("/system/shorten", handlers.CreateShortLink)
}
