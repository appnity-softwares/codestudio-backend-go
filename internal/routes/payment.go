package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterPaymentRoutes(r gin.IRouter) {
	payment := r.Group("/payments")
	payment.Use(middleware.AuthMiddleware())
	{
		payment.POST("/order", handlers.CreateOrder)
		payment.POST("/verify", handlers.VerifyPayment)
	}

	webhooks := r.Group("/webhooks")
	{
		// Webhooks usually verify signature internally, so we might skip AuthMiddleware
		webhooks.POST("/razorpay", handlers.HandleRazorpayWebhook)
	}
}
