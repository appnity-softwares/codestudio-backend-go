package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterArenaRoutes(r gin.IRouter) {
	arena := r.Group("/events")
	{
		// Public (Optional Auth for Registration Status)
		arena.GET("", middleware.OptionalAuthMiddleware(), handlers.ListEvents)
		arena.GET("/:id", middleware.OptionalAuthMiddleware(), handlers.GetEvent)

		// Protected
		protected := arena.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/:id/register", middleware.RequireContestsEnabled(), handlers.RegisterForEvent)
			protected.POST("/:id/rules", middleware.RequireContestsEnabled(), handlers.AcceptRules)
			protected.POST("/:id/join-external", handlers.JoinExternalContest)
			protected.GET("/:id/access", handlers.GetEventAccess) // THE GATEKEEPER

			// Admin only
			admin := protected.Group("/")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.POST("", handlers.CreateEvent)
			}
		}
	}

	// Problem routes - separate group to avoid conflict
	problems := r.Group("/contests")
	{
		// Public problem routes

		// Protected problem routes
		protectedProblems := problems.Group("/")
		protectedProblems.Use(middleware.AuthMiddleware())
		{
			// List & Get Problems (Must be auth to check registration)
			protectedProblems.GET("/:eventId/problems", handlers.ListProblems)
			protectedProblems.GET("/:eventId/problems/:problemId", handlers.GetProblem)

			// Leaderboard
			protectedProblems.GET("/:eventId/leaderboard", handlers.GetContestLeaderboard)

			// Problem submission - requires submissions_enabled
			protectedProblems.POST("/:eventId/problems/:problemId/submit", middleware.RequireSubmissionsEnabled(), handlers.SubmitSolution)
			protectedProblems.POST("/:eventId/problems/:problemId/run", middleware.RequireSubmissionsEnabled(), handlers.RunSolution)
			protectedProblems.GET("/:eventId/problems/:problemId/submissions", handlers.GetUserSubmissions)

			// Admin only - problem management
			adminProblems := protectedProblems.Group("/")
			adminProblems.Use(middleware.AdminMiddleware())
			{
				adminProblems.POST("/:eventId/problems", handlers.CreateProblem)
				adminProblems.PUT("/:eventId/problems/:problemId", handlers.UpdateProblem)
				adminProblems.DELETE("/:eventId/problems/:problemId", handlers.DeleteProblem)
			}
		}
	}
}
