package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/config"
)

func CORSMiddleware() gin.HandlerFunc {
	// Basic CORS setup allowing frontend
	corsConfig := cors.Config{
		AllowOrigins: []string{
			config.AppConfig.FrontendURL,
			"http://localhost:5173",
			"https://codestudio.appnity.cloud", // Production Frontend
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	return cors.New(corsConfig)
}
