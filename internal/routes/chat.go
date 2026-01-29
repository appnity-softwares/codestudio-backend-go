package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func RegisterChatRoutes(r gin.IRouter) {
	chat := r.Group("/chat")
	// Enforce strict auth for chat even if parent group is optional
	chat.Use(middleware.AuthMiddleware(), middleware.FeatureGate(models.SettingFeatureSocialChat, "Direct Messaging"))
	{
		chat.GET("/contacts", handlers.GetContacts)
		chat.GET("/conversations", handlers.GetConversations)
		chat.GET("/messages", handlers.GetMessages) // ?userId=...
		chat.POST("/messages", handlers.SendMessage)
		chat.POST("/read/:senderId", handlers.MarkRead)
	}
}
