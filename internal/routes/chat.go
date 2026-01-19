package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterChatRoutes(r gin.IRouter) {
	chat := r.Group("/chat")
	chat.Use(middleware.AuthMiddleware())
	{
		chat.GET("/contacts", handlers.ListChatContacts)
		chat.GET("/messages", handlers.GetChatHistory)
		chat.POST("/read/:senderId", handlers.MarkMessagesAsRead)
	}
}
