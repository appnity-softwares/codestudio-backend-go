package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterAuthRoutes(r gin.IRouter) {
	r.POST("/register", middleware.RequireRegistrationOpen(), handlers.Register)
	r.POST("/login", handlers.Login)
	// P0 FIX: Protect logout with AuthMiddleware to get claims for revocation
	r.POST("/logout", middleware.AuthMiddleware(), handlers.Logout)

	// OAuth
	r.GET("/google/login", handlers.GoogleLogin)
	r.GET("/google/callback", handlers.GoogleCallback)

	r.GET("/github/login", handlers.GithubLogin)
	r.GET("/github/callback", handlers.GithubCallback)

	// Password Reset
	r.POST("/forgot-password", handlers.ForgotPassword)
	r.POST("/reset-password", handlers.ResetPassword)

	// Utils
	r.GET("/check-username", handlers.CheckUsername)
	r.POST("/appeal", handlers.CreateAppeal)
}
