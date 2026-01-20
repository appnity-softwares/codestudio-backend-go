package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

// RegisterPracticeRoutes sets up the Practice Arena v1 endpoints
// These are separate from official contest routes and have no anti-cheat
func RegisterPracticeRoutes(r gin.IRouter) {
	practice := r.Group("/practice")
	{
		// Public: List problems (with optional auth for solve status)
		practice.GET("/problems", middleware.OptionalAuthMiddleware(), handlers.ListPracticeProblems)
		practice.GET("/problems/:id", middleware.OptionalAuthMiddleware(), handlers.GetPracticeProblem)
		practice.GET("/daily", middleware.OptionalAuthMiddleware(), handlers.GetDailyProblem)

		// Protected: Submit solutions and view history
		protected := practice.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/run", handlers.RunPracticeSolution)
			protected.POST("/submit", handlers.SubmitPracticeSolution)
			protected.GET("/submissions", handlers.GetUserPracticeSubmissions)
		}
	}
}
