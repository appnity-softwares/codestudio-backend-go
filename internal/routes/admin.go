package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
)

func RegisterAdminRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware())

	// Dashboard
	// Dashboard (Generic Staff Access)
	admin.GET("/dashboard", middleware.StaffOnly(""), handlers.AdminGetDashboard)

	// User Management
	users := admin.Group("/users")
	users.Use(middleware.StaffOnly("CanManageUsers"))
	{
		users.GET("", handlers.AdminListUsers)
		users.GET("/search", handlers.AdminSearchUsers)
		users.GET("/:id", handlers.AdminGetUserDetail)
		users.POST("/:id/warn", handlers.AdminWarnUser)
		users.POST("/:id/suspend", handlers.AdminSuspendUser)
		users.POST("/:id/unsuspend", handlers.AdminUnsuspendUser)
		users.POST("/:id/ban-contest", handlers.AdminBanContest)
		users.POST("/:id/trust", handlers.AdminAdjustTrustScore)
		users.PUT("/:id", handlers.AdminUpdateUser)
		users.DELETE("/:id", handlers.AdminDeleteUser)
	}

	// Contest & Problem Management
	contests := admin.Group("")
	contests.Use(middleware.StaffOnly("CanManageContests"))
	{
		contests.GET("/contests", handlers.AdminListContests)
		contests.POST("/contests", handlers.AdminCreateContest)
		contests.PUT("/contests/:id", handlers.AdminUpdateContest)
		contests.DELETE("/contests/:id", handlers.AdminDeleteContest)
		contests.POST("/contests/:id/start", handlers.AdminStartContest)
		contests.POST("/contests/:id/freeze", handlers.AdminFreezeContest)
		contests.POST("/contests/:id/end", handlers.AdminEndContest)
		contests.GET("/contests/:id/participants", handlers.AdminGetContestParticipants)

		// Problems
		contests.GET("/problems/:id", handlers.AdminGetProblem)
		contests.POST("/problems", handlers.AdminCreateProblem)
		contests.PUT("/problems/:id", handlers.AdminUpdateProblem)
		contests.DELETE("/problems/:id", handlers.AdminDeleteProblem)
		contests.POST("/problems/reorder", handlers.AdminReorderProblems)

		// Test Cases
		contests.POST("/problems/:id/testcases", handlers.AdminCreateTestCase)
		contests.PUT("/testcases/:tcId", handlers.AdminUpdateTestCase)
		contests.DELETE("/testcases/:tcId", handlers.AdminDeleteTestCase)

		// Practice Problems
		contests.GET("/practice-problems", handlers.AdminListPracticeProblems)
		contests.GET("/practice-problems/:id", handlers.AdminGetPracticeProblem)
		contests.POST("/practice-problems", handlers.AdminCreatePracticeProblem)
		contests.PUT("/practice-problems/:id", handlers.AdminUpdatePracticeProblem)
		contests.DELETE("/practice-problems/:id", handlers.AdminDeletePracticeProblem)
	}

	// Flag Review & Submissions (Moderation)
	moderation := admin.Group("")
	moderation.Use(middleware.StaffOnly("CanManageSnippets")) // Snippets/Submissions grouped
	{
		moderation.GET("/flags", handlers.AdminGetFlags)
		moderation.POST("/flags/:id/ignore", handlers.AdminIgnoreFlag)
		moderation.POST("/flags/:id/warn", handlers.AdminWarnSubmission)
		moderation.POST("/flags/:id/disqualify-submission", handlers.AdminDisqualifySubmission)
		moderation.POST("/flags/:id/disqualify-user", handlers.AdminDisqualifyUser)

		moderation.GET("/submissions", handlers.AdminListSubmissions)
		moderation.GET("/submissions/:id", handlers.AdminGetSubmissionDetail)
		moderation.POST("/submissions/:id/restore", handlers.AdminRestoreSubmission)

		moderation.POST("/snippets/:id/pin", handlers.AdminPinSnippet)

		// Avatars also usually staff/moderation
		moderation.POST("/avatars", handlers.AdminAddAvatarSeed)
		moderation.DELETE("/avatars/:id", handlers.AdminDeleteAvatarSeed)
	}

	// Audit Logs (Granular Permission)
	audit := admin.Group("")
	audit.Use(middleware.StaffOnly("CanViewAuditLogs"))
	{
		audit.GET("/audit-logs", handlers.AdminGetAuditLogs)
		audit.GET("/analytics/suspicious", handlers.AdminGetSuspiciousActivity)
	}

	// System & Admin-Only Stuff
	restricted := admin.Group("")
	restricted.Use(middleware.AdminOnly())
	{
		// Role Permissions
		restricted.GET("/roles/permissions", handlers.AdminGetRolePermissions)
		restricted.PUT("/roles/permissions", handlers.AdminUpdateRolePermission)

		// System Settings
		restricted.GET("/system", handlers.AdminGetSystemSettings)
		restricted.PUT("/system", handlers.AdminUpdateSystemSettings)

		// Analytics (Full)
		restricted.GET("/analytics/top-snippets", handlers.AdminGetTopSnippets)

		// Changelog
		restricted.GET("/changelog", handlers.AdminListChangelogs)
		restricted.POST("/changelog", handlers.AdminCreateChangelog)
		restricted.PUT("/changelog/:id", handlers.AdminUpdateChangelog)
		restricted.DELETE("/changelog/:id", handlers.AdminDeleteChangelog)

		// Feedback
		restricted.GET("/feedback", handlers.AdminListFeedback)
		restricted.PUT("/feedback/:id/status", handlers.AdminUpdateFeedbackStatus)
		restricted.POST("/feedback/:id/lock", handlers.AdminLockFeedback)
		restricted.POST("/feedback/:id/hide", handlers.AdminHideFeedback)
		restricted.POST("/feedback/:id/pin", handlers.AdminPinFeedback)
		restricted.POST("/feedback/:id/convert", handlers.AdminConvertToChangelog)
	}
}
