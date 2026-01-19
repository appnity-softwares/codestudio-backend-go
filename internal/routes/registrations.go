package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterRegistrationRoutes(r gin.IRouter) {
	registrations := r.Group("/registrations")
	registrations.Use(middleware.AuthMiddleware())
	{

		registrations.GET("/my", handlers.GetMyRegistrations)

		// Admin Routes
		admin := registrations.Group("/")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("", handlers.ListRegistrations)
			admin.PATCH("/:id/status", handlers.UpdateRegistrationStatus)
		}
	}
}
