package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterPlaylistRoutes(r *gin.RouterGroup) {
	playlists := r.Group("/playlists")
	{
		playlists.GET("", handlers.ListPlaylists)
		playlists.GET("/:id", handlers.GetPlaylist)

		// Auth required for modification
		protected := playlists.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("", handlers.CreatePlaylist)
			protected.PUT("/:id", handlers.UpdatePlaylist)
			protected.DELETE("/:id", handlers.DeletePlaylist)
			protected.POST("/:id/snippets", handlers.AddSnippetToPlaylist)
			protected.DELETE("/:id/snippets/:snippetId", handlers.RemoveSnippetFromPlaylist)
			protected.POST("/:id/reorder", handlers.ReorderPlaylist)
			protected.POST("/:id/claim", handlers.ClaimEndorsement)
		}
	}
}
