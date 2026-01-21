package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	// 0. Load Config & Initialize Logger
	config.LoadConfig()

	// Environment-based logger initialization (production = JSON, development = pretty)
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}
	logger.Init(env)

	logger.Info().Str("environment", env).Msg("Starting CodeStudio Backend...")

	// Set Gin mode based on environment
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 2. Connect Database
	database.Connect()
	database.InitRedis() // Initialize Redis

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
		&models.FeedbackMessage{},
		&models.FeedbackReaction{},
		&models.FeedbackDisagree{},
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
	r := gin.Default()

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

		// Public system status (for maintenance page)
		api.GET("/system/status", handlers.PublicGetSystemStatus)

		// Protected routes - apply maintenance mode check
		protected := api.Group("")
		protected.Use(middleware.OptionalAuthMiddleware(), middleware.MaintenanceMode())

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
		routes.RegisterFeedbackRoutes(api)       // Feedback Wall Routes (Hybrid Public/Protected)
	}

	// Enhanced health check with DB and Redis status
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		redisStatus := "ok"

		// Check database connection
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "error"
		}

		// Check Redis connection
		if database.Redis != nil {
			if _, err := database.Redis.Ping(context.Background()).Result(); err != nil {
				redisStatus = "error"
			}
		} else {
			redisStatus = "not configured"
		}

		status := "ok"
		if dbStatus != "ok" || (redisStatus != "ok" && redisStatus != "not configured") {
			status = "degraded"
		}

		c.JSON(200, gin.H{
			"status":  status,
			"message": "CodeStudio Backend is running 🚀",
			"checks": gin.H{
				"database": dbStatus,
				"redis":    redisStatus,
			},
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

	// 6. Start Server with graceful shutdown
	port := config.AppConfig.Port
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info().Str("port", port).Str("env", env).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("🛑 Shutting down server gracefully...")

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("✅ Server exited gracefully")
}
