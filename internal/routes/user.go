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
			protected.POST("/equip-aura", handlers.EquipAura)
			protected.POST("/equip-theme", handlers.EquipTheme)
			protected.POST("/github/sync", handlers.SyncGithubStats)
			protected.PUT("/github/settings", handlers.UpdateGithubStatsSettings)
		}

		// Public (Wildcard last)
		// users.GET("", handlers.ListUsers) // Community list disabled for MVP
		users.GET("/profile/summary", handlers.GetProfileSummary) // /users/profile/summary
		users.GET("/avatars", handlers.GetAvatarSeeds)
		users.GET("/:username", handlers.GetProfile)
		users.GET("/:username/snippets", handlers.GetUserSnippets)
		users.GET("/:username/badges", handlers.GetBadges)

		// History (Authenticated)
		users.GET("/me/contests", middleware.AuthMiddleware(), handlers.GetMyContestHistory)

		// Onboarding (Authenticated)
		users.POST("/onboarding", middleware.AuthMiddleware(), handlers.CompleteOnboarding)
		users.POST("/spend-xp", middleware.AuthMiddleware(), handlers.SpendXP)

		// Vault Handshake
		users.POST("/vault/key", middleware.AuthMiddleware(), handlers.GenerateVaultKey)
		users.POST("/vault/verify", handlers.VerifyVaultKey)
	}

	// Community & Public Profile Routes (Root under /api usually)
	r.GET("/community/users", middleware.OptionalAuthMiddleware(), handlers.ListCommunityUsers)
	r.GET("/community/search-suggestions", handlers.SearchSuggestions)
	r.GET("/public/users/:username", handlers.GetPublicProfile)
	r.GET("/leaderboard/global", handlers.GetGlobalLeaderboard)
}
