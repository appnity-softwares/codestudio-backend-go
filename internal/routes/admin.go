package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterAdminRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())

	// Dashboard
	admin.GET("/dashboard", handlers.AdminGetDashboard)

	// Contest Management
	// Contest Management
	admin.GET("/contests", handlers.AdminListContests)
	admin.POST("/contests", handlers.AdminCreateContest)
	admin.PUT("/contests/:id", handlers.AdminUpdateContest)
	admin.DELETE("/contests/:id", handlers.AdminDeleteContest)
	admin.POST("/contests/:id/start", handlers.AdminStartContest)
	admin.POST("/contests/:id/freeze", handlers.AdminFreezeContest)
	admin.POST("/contests/:id/end", handlers.AdminEndContest)
	admin.GET("/contests/:id/participants", handlers.AdminGetContestParticipants)

	// Problem Management
	admin.GET("/problems/:id", handlers.AdminGetProblem)
	admin.POST("/problems", handlers.AdminCreateProblem)
	admin.PUT("/problems/:id", handlers.AdminUpdateProblem)
	admin.DELETE("/problems/:id", handlers.AdminDeleteProblem)
	admin.POST("/problems/reorder", handlers.AdminReorderProblems)

	// Test Case Management
	admin.POST("/problems/:id/testcases", handlers.AdminCreateTestCase)
	admin.PUT("/testcases/:tcId", handlers.AdminUpdateTestCase)
	admin.DELETE("/testcases/:tcId", handlers.AdminDeleteTestCase)

	// Flag Review
	admin.GET("/flags", handlers.AdminGetFlags)
	admin.POST("/flags/:id/ignore", handlers.AdminIgnoreFlag)
	admin.POST("/flags/:id/warn", handlers.AdminWarnSubmission)
	admin.POST("/flags/:id/disqualify-submission", handlers.AdminDisqualifySubmission)
	admin.POST("/flags/:id/disqualify-user", handlers.AdminDisqualifyUser)

	// User Management
	admin.GET("/users", handlers.AdminListUsers)
	admin.GET("/users/search", handlers.AdminSearchUsers)
	admin.GET("/users/:id", handlers.AdminGetUserDetail)
	admin.POST("/users/:id/warn", handlers.AdminWarnUser)
	admin.POST("/users/:id/suspend", handlers.AdminSuspendUser)
	admin.POST("/users/:id/unsuspend", handlers.AdminUnsuspendUser)
	admin.POST("/users/:id/ban-contest", handlers.AdminBanContest)
	admin.POST("/users/:id/trust", handlers.AdminAdjustTrustScore)

	// Submissions
	admin.GET("/submissions", handlers.AdminListSubmissions)
	admin.GET("/submissions/:id", handlers.AdminGetSubmissionDetail)
	admin.POST("/submissions/:id/restore", handlers.AdminRestoreSubmission)

	// Snippet Moderation
	admin.POST("/snippets/:id/pin", handlers.AdminPinSnippet)

	// System Settings
	admin.GET("/system", handlers.AdminGetSystemSettings)
	admin.PUT("/system", handlers.AdminUpdateSystemSettings)

	// Analytics
	admin.GET("/analytics/top-snippets", handlers.AdminGetTopSnippets)
	admin.GET("/analytics/suspicious", handlers.AdminGetSuspiciousActivity)

	// Audit
	admin.GET("/audit-logs", handlers.AdminGetAuditLogs)

	// Changelog
	admin.GET("/changelog", handlers.AdminListChangelogs)
	admin.POST("/changelog", handlers.AdminCreateChangelog)
	admin.PUT("/changelog/:id", handlers.AdminUpdateChangelog)
	admin.DELETE("/changelog/:id", handlers.AdminDeleteChangelog)
}
