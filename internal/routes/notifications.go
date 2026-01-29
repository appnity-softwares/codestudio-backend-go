package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterNotificationRoutes(r gin.IRouter) {
	notifications := r.Group("/notifications")
	notifications.Use(middleware.AuthMiddleware())
	{
		notifications.GET("", handlers.GetNotifications)
		notifications.GET("/unread-count", handlers.GetUnreadCount)
		notifications.PUT("/:id/read", handlers.MarkNotificationRead)
		notifications.PUT("/read-all", handlers.MarkAllNotificationsRead)
		notifications.DELETE("/:id", handlers.DeleteNotification)
	}
}
