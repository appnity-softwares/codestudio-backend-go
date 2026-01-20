package main

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/internal/routes"
	"github.com/pushp314/devconnect-backend/pkg/logger"
)

func main() {
	// 0. Initialize Logger
	config.LoadConfig()
	logger.Init("development")

	logger.Info().Msg("Starting DevConnect Backend...")

	// 2. Connect Database
	database.Connect()

	// --- Database Migration Stage ---
	logger.Info().Msg("🔄 Running Database Migrations (Stage 1: Tables)...")

	// Temporarily disable foreign key constraints to handle circular dependencies (User <-> Snippet)
	database.DB.Config.DisableForeignKeyConstraintWhenMigrating = true

	tableModels := []interface{}{
		&models.User{},
		&models.Event{},
		&models.Snippet{},
		&models.Message{},
		&models.Registration{},
		&models.Submission{},
		&models.Problem{},
		&models.TestCase{},
		&models.ChangelogEntry{},
		&models.SubmissionFlag{},
		&models.SubmissionMetrics{},
		&models.PracticeProblem{},
		&models.PracticeSubmission{},
		&models.AdminAction{},
		&models.UserSuspension{},
		&models.TrustScoreHistory{},
		&models.SystemSettings{},
		&models.AdminAuditLog{},
		&models.EntityView{},
		&models.EntityCopy{},
	}

	for _, m := range tableModels {
		if err := database.DB.AutoMigrate(m); err != nil {
			logger.Fatal().Err(err).Msgf("Failed to migrate table for %T", m)
		}
	}

	logger.Info().Msg("🔄 Running Database Migrations (Stage 2: Constraints)...")
	// Re-enable and run again to add all foreign key constraints
	database.DB.Config.DisableForeignKeyConstraintWhenMigrating = false
	if err := database.DB.AutoMigrate(tableModels...); err != nil {
		logger.Fatal().Err(err).Msg("Failed to add database constraints")
	}
	logger.Info().Msg("✅ Database Migrations Complete")

	// 3. Init OAuth
	handlers.InitOAuthConfig()

	// 4. Setup Router
	r := gin.New()

	// Middlewares
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.ErrorHandlerMiddleware())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// Exempt /socket.io from rate limiting
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/socket.io/" || len(c.Request.URL.Path) > 10 && c.Request.URL.Path[:10] == "/socket.io/" {
			c.Next()
			return
		}
		middleware.GeneralRateLimit()(c)
	})

	// 5. Register Routes
	api := r.Group("/api")
	{
		// Auth routes - no maintenance check (allow login even during maintenance)
		auth := api.Group("/auth")
		auth.Use(middleware.AuthRateLimit())
		routes.RegisterAuthRoutes(auth)

		// Protected routes - apply maintenance mode check
		protected := api.Group("")
		protected.Use(middleware.MaintenanceMode())

		routes.RegisterSnippetRoutes(protected)
		routes.RegisterUserRoutes(protected)
		routes.RegisterArenaRoutes(protected)
		routes.RegisterUploadRoutes(protected)
		routes.RegisterRegistrationRoutes(protected)
		routes.RegisterPaymentRoutes(protected)
		routes.RegisterChatRoutes(protected)
		routes.SetupChangelogRoutes(api)         // Public changelog - no maintenance check
		routes.RegisterAdminRoutes(api)          // Admin routes bypass maintenance
		routes.RegisterPracticeRoutes(protected) // v1.2: Practice Arena
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "DevConnect Go Backend is running 🚀",
		})
	})

	// Sitemap & SEO
	r.GET("/sitemap.xml", handlers.GenerateSitemap)
	r.GET("/robots.txt", handlers.GenerateRobotsTXT)

	// Init Socket.io
	socketServer := handlers.InitSocketServer()
	defer socketServer.Close()

	// Register Socket.io Routes
	r.GET("/socket.io/*any", handlers.SocketHandler(socketServer))
	r.POST("/socket.io/*any", handlers.SocketHandler(socketServer))

	// 6. Start Server
	port := config.AppConfig.Port
	if port == "" {
		port = "8080"
	}
	logger.Info().Str("port", port).Msg("Server starting")
	if err := r.Run(":" + port); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
