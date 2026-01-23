package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterSnippetRoutes(r gin.IRouter) {
	snippets := r.Group("/snippets")
	{
		snippets.GET("", middleware.OptionalAuthMiddleware(), handlers.ListSnippets)
		snippets.GET("/:id", middleware.OptionalAuthMiddleware(), handlers.GetSnippet)
		snippets.GET("/:id/similar", handlers.GetSimilarSnippets)
		snippets.POST("/:id/run", middleware.OptionalAuthMiddleware(), handlers.RunSnippet)
		snippets.POST("/execute", middleware.ExecuteRateLimit(), handlers.ExecuteCode)

		// Protected Base (Auth Only)
		protected := snippets.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Read-Only / Tracking (Allowed even if creation is disabled)
			protected.POST("/:id/copy", handlers.RecordSnippetCopy)
			protected.POST("/:id/view", handlers.RecordSnippetView)

			// Mutative / Creation Actions (Subject to System Switch)
			creationEnabled := protected.Group("")
			creationEnabled.Use(middleware.RequireSnippetsEnabled())
			{
				creationEnabled.POST("", handlers.CreateSnippet)
				creationEnabled.PUT("/:id", handlers.UpdateSnippet)
				creationEnabled.DELETE("/:id", handlers.DeleteSnippet)
				creationEnabled.PATCH("/:id/output", handlers.UpdateSnippetOutput)
				creationEnabled.POST("/:id/publish", handlers.PublishSnippet)
				creationEnabled.POST("/:id/fork", handlers.ForkSnippet)
			}
		}
	}

	// v1.2: Smart Feed (Public)
	r.GET("/feed", middleware.OptionalAuthMiddleware(), handlers.GetFeed)
}
