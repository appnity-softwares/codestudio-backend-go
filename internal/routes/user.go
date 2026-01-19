package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterUserRoutes(r gin.IRouter) {
	users := r.Group("/users")
	{
		// Protected (Specific paths first)
		protected := users.Group("/profile")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/stats", handlers.GetStats)
			protected.GET("", handlers.GetProfile)
			protected.PUT("", handlers.UpdateProfile)
		}

		// Public (Wildcard last)
		// users.GET("", handlers.ListUsers) // Community list disabled for MVP
		users.GET("/profile/summary", handlers.GetProfileSummary)
		users.GET("/:username", handlers.GetProfile)
		users.GET("/:username/snippets", handlers.GetUserSnippets)
	}
}
