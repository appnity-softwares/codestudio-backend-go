package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
)

// SetupChangelogRoutes configures public changelog routes
func SetupChangelogRoutes(r *gin.RouterGroup) {
	r.GET("/changelog", handlers.ListChangelog)
}
