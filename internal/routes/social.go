package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func RegisterSocialRoutes(r gin.IRouter) {
	social := r.Group("/users")
	{
		// Authenticated Social Actions
		protected := social.Group("")
		protected.Use(middleware.AuthMiddleware(), middleware.FeatureGate(models.SettingFeatureSocialFollow, "Follower System"))
		{
			protected.POST("/:username/link", handlers.LinkUser)
			protected.DELETE("/:username/link", handlers.UnlinkUser)
			protected.GET("/:username/link/status", handlers.CheckLinkStatus)

			// v1.3: Link Requests (Private Accounts)
			protected.GET("/link-requests", handlers.ListLinkRequests)
			protected.POST("/link-requests/:id/accept", handlers.AcceptLinkRequest)
			protected.POST("/link-requests/:id/reject", handlers.RejectLinkRequest)

			// v1.3: Safety & Moderation
			protected.POST("/:username/block", handlers.BlockUser)
			protected.DELETE("/:username/block", handlers.UnblockUser)
			protected.GET("/blocks", handlers.GetBlockedUsers) // Using /users/blocks since it's under /users group
			protected.POST("/report", handlers.ReportTarget)
		}

		// Public Social Data (or Optional Auth if needed, but simple public for now)
		social.GET("/:username/linkers", handlers.GetLinkers)
		social.GET("/:username/linked", handlers.GetLinked)
	}

	snippet := r.Group("/snippets")
	{
		// Authenticated Snippet Engagement
		protected := snippet.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/:id/like", handlers.ToggleLikeSnippet)
			protected.POST("/:id/dislike", handlers.ToggleDislikeSnippet)
			protected.GET("/:id/like", handlers.CheckSnippetLike) // Auth check for 'my' like status
			protected.POST("/:id/comments", handlers.AddComment)
		}

		// Public Snippet Data
		snippet.GET("/:id/comments", handlers.GetSnippetComments)
	}

	comments := r.Group("/comments")
	{
		protected := comments.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.DELETE("/:id", handlers.DeleteComment)
		}
	}
}
