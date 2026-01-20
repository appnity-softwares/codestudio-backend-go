package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterFeedbackRoutes(r *gin.RouterGroup) {
	feedback := r.Group("/feedback")

	// Public View (Optional Auth for React state)
	feedback.GET("", middleware.OptionalAuthMiddleware(), handlers.GetFeedback)

	// Protected Actions
	protected := feedback.Group("")
	protected.Use(middleware.AuthMiddleware()) // Strict Auth
	{
		protected.POST("", handlers.CreateFeedback)
		protected.POST("/:id/react", handlers.ReactFeedback)
		protected.POST("/:id/disagree", handlers.DisagreeFeedback)
	}
}
