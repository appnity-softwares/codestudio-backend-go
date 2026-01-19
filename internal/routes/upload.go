package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterUploadRoutes(r gin.IRouter) {
	upload := r.Group("/upload")
	upload.Use(middleware.AuthMiddleware())
	{
		upload.POST("/profile-image", handlers.UploadProfileImage)
		upload.POST("/chat-attachment", handlers.UploadChatAttachment)

		// Generic
		upload.POST("/", handlers.UploadFile)
	}
}
