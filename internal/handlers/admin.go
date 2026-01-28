package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// Helper: Log Admin Action
func logAdminAction(tx *gorm.DB, adminID string, action models.ActionType, targetID string, targetType string, reason string) error {
	audit := models.AdminAction{
		ID:         uuid.New().String(),
		AdminID:    adminID,
		Action:     action,
		TargetID:   targetID,
		TargetType: targetType,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}
	return tx.Create(&audit).Error
}

func getAdminID(c *gin.Context) string {
	val, exists := c.Get("userId")
	if !exists {
		return ""
	}
	return val.(string)
}

// --- Contest Management ---
// Moved to admin_contest.go

// --- Flag Review ---

func AdminGetFlags(c *gin.Context) {
	var submissions []models.Submission
	if err := database.DB.Preload("User").Preload("Problem").Preload("Flags").
		Joins("JOIN submission_flags ON submission_flags.submission_id = submissions.id").
		Group("submissions.id").
		Order("submissions.created_at desc").
		Find(&submissions).Error; err != nil {
		c.JSON(500, gin.H{"error": "Fetch Failed"})
		return
	}
	c.JSON(200, gin.H{"submissions": submissions})
}

func AdminIgnoreFlag(c *gin.Context) {
	flagID := c.Param("id") // Here expecting Submission ID actually based on routes, or specific flag?
	// Route is /admin/flags/:id/ignore. Let's assume ID is Submission ID for MVP simplicity or Flag ID.
	// If ID is Submission ID, we ignore all flags for that submission.

	adminID := getAdminID(c)
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		return logAdminAction(tx, adminID, models.ActionIgnoreFlag, flagID, "flag", "Ignored")
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Flag Ignored"})
}

func AdminWarnSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	adminID := getAdminID(c)
	var req struct {
		Reason string `json:"reason"`
	}
	c.BindJSON(&req)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var sub models.Submission
		if err := tx.First(&sub, "id = ?", submissionID).Error; err != nil {
			return err
		}

		flag := models.SubmissionFlag{
			ID:           uuid.New().String(),
			SubmissionID: submissionID,
			Type:         models.FlagTypeSuspicious,
			Details:      "WARN: " + req.Reason,
			CreatedAt:    time.Now(),
		}
		tx.Create(&flag)

		tx.Model(&models.User{}).Where("id = ?", sub.UserID).
			Update("trust_score", gorm.Expr("GREATEST(trust_score - 10, 0)"))

		return logAdminAction(tx, adminID, models.ActionWarnSubmission, submissionID, "submission", req.Reason)
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Warned"})
}

func AdminDisqualifySubmission(c *gin.Context) {
	submissionID := c.Param("id")
	adminID := getAdminID(c)
	var req struct {
		Reason string `json:"reason"`
	}
	c.BindJSON(&req)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var sub models.Submission
		if err := tx.First(&sub, "id = ?", submissionID).Error; err != nil {
			return err
		}

		tx.Model(&sub).Updates(map[string]interface{}{"status": "DISQUALIFIED", "verdict": "Disqualified by Admin"})

		flag := models.SubmissionFlag{
			ID:           uuid.New().String(),
			SubmissionID: submissionID,
			Type:         models.FlagTypeSuspicious,
			Details:      "DQ: " + req.Reason,
			CreatedAt:    time.Now(),
		}
		tx.Create(&flag)

		tx.Model(&models.User{}).Where("id = ?", sub.UserID).
			Update("trust_score", gorm.Expr("GREATEST(trust_score - 50, 0)"))

		return logAdminAction(tx, adminID, models.ActionDisqualifySub, submissionID, "submission", req.Reason)
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Disqualified"})
}

func AdminDisqualifyUser(c *gin.Context) {
	// /admin/flags/:id/disqualify-user
	// :id here is Submission ID context
	submissionID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var sub models.Submission
		if err := tx.First(&sub, "id = ?", submissionID).Error; err != nil {
			return err
		}

		// 1. DQ Submission
		tx.Model(&sub).Updates(map[string]interface{}{"status": "DISQUALIFIED", "verdict": "DQ & User Ban"})

		// 2. Nuke Trust Score
		tx.Model(&models.User{}).Where("id = ?", sub.UserID).Update("trust_score", 0)

		// 3. Mark Registration as DQ (if exists)
		tx.Model(&models.Registration{}).Where("user_id = ? AND event_id = ?", sub.UserID, sub.EventID).Update("status", "BANNED")

		return logAdminAction(tx, adminID, models.ActionBanUser, sub.UserID, "user", "Banned via Flag Review")
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "User Disqualified"})
}

// --- User Moderation ---

func AdminGetUser(c *gin.Context) {
	userID := c.Param("id")
	var user models.User
	if err := database.DB.Preload("Submissions").First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Not Found"})
		return
	}
	c.JSON(200, gin.H{"user": user})
}

func AdminWarnUser(c *gin.Context) {
	userID := c.Param("id")
	adminID := getAdminID(c)
	var req struct {
		Reason string `json:"reason"`
	}
	c.BindJSON(&req)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		tx.Model(&models.User{}).Where("id = ?", userID).Update("trust_score", gorm.Expr("GREATEST(trust_score - 5, 0)"))
		return logAdminAction(tx, adminID, models.ActionWarnUser, userID, "user", req.Reason)
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "User Warned"})
}

func AdminBanContest(c *gin.Context) {
	// /admin/users/:id/ban-contest
	// Body should have contest_id
	userID := c.Param("id")
	adminID := getAdminID(c)
	var req struct {
		EventID string `json:"eventId"`
	}
	c.BindJSON(&req)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Update registration
		if err := tx.Model(&models.Registration{}).
			Where("user_id = ? AND event_id = ?", userID, req.EventID).
			Update("status", "BANNED").Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionBanUser, userID, "user", "Banned from contest "+req.EventID)
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "User Banned from Contest"})
}

func AdminGetAuditLogs(c *gin.Context) {
	var logs []models.AdminAction
	database.DB.Preload("Admin").Order("created_at desc").Limit(100).Find(&logs)
	c.JSON(200, gin.H{"logs": logs})
}

// ============================================
// v1.2: SNIPPET & USER MODERATION
// ============================================

// AdminPinSnippet toggles the isFeatured flag for a snippet (Editor Picks)
func AdminPinSnippet(c *gin.Context) {
	snippetID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Featured bool `json:"featured"`
	}
	c.BindJSON(&req)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Snippet{}).
			Where("id = ?", snippetID).
			Update("is_featured", req.Featured).Error; err != nil {
			return err
		}

		action := "Unpinned from Editor Feed"
		if req.Featured {
			action = "Pinned to Editor Feed"
		}
		return logAdminAction(tx, adminID, models.ActionPinSnippet, snippetID, "snippet", action)
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Snippet pin status updated", "featured": req.Featured})
}

// AdminGetSnippets lists snippets for admin/staff
func AdminGetSnippets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Snippet{}).Preload("Author")

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR language ILIKE ?", searchPattern, searchPattern)
	}

	var total int64
	query.Count(&total)

	var snippets []models.Snippet
	if err := query.Order("\"createdAt\" desc").Offset(offset).Limit(limit).Find(&snippets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch snippets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snippets": snippets,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// AdminDeleteSnippet allows staff to delete any snippet
func AdminDeleteSnippet(c *gin.Context) {
	snippetID := c.Param("id")
	adminID := getAdminID(c)

	var snippet models.Snippet
	if err := database.DB.First(&snippet, "id = ?", snippetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	if err := database.DB.Delete(&snippet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete snippet"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionDeleteSnippet, snippetID, "snippet", "Deleted by staff")

	c.JSON(http.StatusOK, gin.H{"message": "Snippet deleted successfully"})
}

// AdminAdjustTrustScore manually sets a user's trust score with audit logging
func AdminAdjustTrustScore(c *gin.Context) {
	userID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		TrustScore int    `json:"trustScore"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Clamp trust score to valid range
	if req.TrustScore < 0 {
		req.TrustScore = 0
	}
	if req.TrustScore > 100 {
		req.TrustScore = 100
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).
			Where("id = ?", userID).
			Update("trust_score", req.TrustScore).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionAdjustTrust, userID, "user", req.Reason)
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Trust score updated", "trustScore": req.TrustScore})
}

// AdminGrantUserXP allows admins to manually grant or deduct XP
func AdminGrantUserXP(c *gin.Context) {
	userID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Amount int    `json:"amount"` // Positive to grant, negative to deduct
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Update User XP
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("xp", gorm.Expr("GREATEST(xp + ?, 0)", req.Amount)).Error; err != nil {
			return err
		}

		// Log Action
		action := "Granted " + strconv.Itoa(req.Amount) + " XP"
		if req.Amount < 0 {
			action = "Deducted " + strconv.Itoa(-req.Amount) + " XP"
		}

		// Ideally we should also insert into an XP Ledger/Transaction table if we have one.
		// For now, Admin Action Log is sufficient for audit.

		return logAdminAction(tx, adminID, models.ActionUpdateUser, userID, "user", action+": "+req.Reason)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "XP updated successfully"})
}

// ============================================
// DASHBOARD METRICS
// ============================================

// AdminGetDashboard returns high-level metrics for the admin dashboard
func AdminGetDashboard(c *gin.Context) {
	var metrics models.DashboardMetrics

	// Total Users
	database.DB.Model(&models.User{}).Count(&metrics.TotalUsers)

	// Active Users Today (users with submissions today)
	today := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&models.Submission{}).
		Where("created_at >= ?", today).
		Distinct("user_id").
		Count(&metrics.ActiveUsersToday)

	// Total Snippets
	database.DB.Model(&models.Snippet{}).Count(&metrics.TotalSnippets)

	// Total Contests
	database.DB.Model(&models.Event{}).Count(&metrics.TotalContests)

	// Live Contests
	database.DB.Model(&models.Event{}).Where("status = ?", models.EventStatusLive).Count(&metrics.LiveContests)

	// Flagged Submissions (submissions with at least one flag)
	database.DB.Model(&models.Submission{}).
		Joins("JOIN submission_flags ON submission_flags.submission_id = submissions.id").
		Distinct("submissions.id").
		Count(&metrics.FlaggedSubmissions)

	// Low Trust Users (trust_score < 50)
	database.DB.Model(&models.User{}).Where("trust_score < ?", 50).Count(&metrics.LowTrustUsers)

	// Total Submissions
	database.DB.Model(&models.Submission{}).Count(&metrics.TotalSubmissions)

	// Pending Submissions
	database.DB.Model(&models.Submission{}).Where("status = ?", models.SubStatusPending).Count(&metrics.PendingSubmissions)

	// Suspended Users (active suspensions)
	database.DB.Model(&models.UserSuspension{}).
		Where("lifted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", time.Now()).
		Distinct("user_id").
		Count(&metrics.SuspendedUsers)

	// v1.3: Enhanced Metrics
	database.DB.Model(&models.Snippet{}).Where("created_at >= ?", today).Count(&metrics.NewSnippetsToday)

	var totalSubs, acceptedSubs int64
	database.DB.Model(&models.Submission{}).Count(&totalSubs)
	if totalSubs > 0 {
		database.DB.Model(&models.Submission{}).Where("status = ?", "ACCEPTED").Count(&acceptedSubs)
		metrics.SubmissionSuccessRate = float64(acceptedSubs) / float64(totalSubs)
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// ============================================
// USER MANAGEMENT
// ============================================

// AdminListUsers returns a paginated list of all users
func AdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search") // Unified search param

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	users := []models.User{} // Initialize as empty slice, not nil
	var total int64

	query := database.DB.Model(&models.User{})

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("email ILIKE ? OR username ILIKE ? OR id = ?", searchPattern, searchPattern, search)
	}

	query.Count(&total)
	query.Order("\"createdAt\" desc").Offset(offset).Limit(limit).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// AdminSearchUsers searches users by email, username, or ID
func AdminSearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	var users []models.User
	searchPattern := "%" + query + "%"

	database.DB.Where("email ILIKE ? OR username ILIKE ? OR id = ?", searchPattern, searchPattern, query).
		Limit(50).
		Find(&users)

	c.JSON(http.StatusOK, gin.H{"users": users, "count": len(users)})
}

// AdminGetUserDetail returns detailed user info including history
func AdminGetUserDetail(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get suspensions
	var suspensions []models.UserSuspension
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(10).Find(&suspensions)

	// Get trust score history
	var trustHistory []models.TrustScoreHistory
	database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&trustHistory)

	// Get recent submissions
	var submissions []models.Submission
	database.DB.Preload("Flags").Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&submissions)

	// Get unique IPs
	var ips []string
	database.DB.Model(&models.SubmissionMetrics{}).
		Joins("JOIN submissions ON submissions.id = submission_metrics.submission_id").
		Where("submissions.user_id = ?", userID).
		Distinct("submission_metrics.ip").
		Pluck("submission_metrics.ip", &ips)

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"suspensions":  suspensions,
		"trustHistory": trustHistory,
		"submissions":  submissions,
		"ips":          ips,
	})
}

// AdminSuspendUser suspends a user account
func AdminSuspendUser(c *gin.Context) {
	userID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Type      string `json:"type" binding:"required"` // TEMPORARY or PERMANENT
		Reason    string `json:"reason" binding:"required"`
		ExpiresIn int    `json:"expiresIn"` // Hours, for temporary
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	suspensionType := models.SuspensionTemporary
	if req.Type == "PERMANENT" {
		suspensionType = models.SuspensionPermanent
	}

	var expiresAt *time.Time
	if suspensionType == models.SuspensionTemporary && req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		expiresAt = &exp
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		suspension := models.UserSuspension{
			ID:        uuid.New().String(),
			UserID:    userID,
			AdminID:   adminID,
			Type:      suspensionType,
			Reason:    req.Reason,
			ExpiresAt: expiresAt,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&suspension).Error; err != nil {
			return err
		}

		// Also block the user
		tx.Model(&models.User{}).Where("id = ?", userID).Update("is_blocked", true)

		return logAdminAction(tx, adminID, models.ActionBanUser, userID, "user", req.Reason)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User suspended"})
}

// AdminUnsuspendUser lifts a user suspension
func AdminUnsuspendUser(c *gin.Context) {
	userID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Lift all active suspensions
		now := time.Now()
		tx.Model(&models.UserSuspension{}).
			Where("user_id = ? AND lifted_at IS NULL", userID).
			Updates(map[string]interface{}{"lifted_at": now, "lifted_by": adminID})

		// Unblock user
		tx.Model(&models.User{}).Where("id = ?", userID).Update("is_blocked", false)

		return logAdminAction(tx, adminID, models.ActionWarnUser, userID, "user", "Suspension lifted")
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User unsuspended"})
}

// ============================================
// SUBMISSION MANAGEMENT
// ============================================

// AdminListSubmissions returns filtered, paginated submissions
func AdminListSubmissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Filters
	contestID := c.Query("contestId")
	userID := c.Query("userId")
	verdict := c.Query("verdict")
	flagged := c.Query("flagged")

	query := database.DB.Model(&models.Submission{}).Preload("User").Preload("Problem").Preload("Flags")

	if contestID != "" {
		query = query.Where("event_id = ?", contestID)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if verdict != "" {
		query = query.Where("status = ?", verdict)
	}
	if flagged == "true" {
		query = query.Joins("JOIN submission_flags ON submission_flags.submission_id = submissions.id").
			Group("submissions.id")
	}

	var total int64
	query.Count(&total)

	var submissions []models.Submission
	query.Order("created_at desc").Offset(offset).Limit(limit).Find(&submissions)

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// AdminGetSubmissionDetail returns full submission details
func AdminGetSubmissionDetail(c *gin.Context) {
	submissionID := c.Param("id")

	var submission models.Submission
	if err := database.DB.Preload("User").Preload("Problem").Preload("Flags").Preload("Metrics").
		First(&submission, "id = ?", submissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submission": submission})
}

// AdminRestoreSubmission restores a disqualified submission
func AdminRestoreSubmission(c *gin.Context) {
	submissionID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Reason string `json:"reason"`
	}
	c.BindJSON(&req)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Restore to pending for re-evaluation or set to a specific status
		tx.Model(&models.Submission{}).Where("id = ?", submissionID).
			Updates(map[string]interface{}{"status": models.SubStatusPending, "verdict": "Restored by Admin"})

		return logAdminAction(tx, adminID, models.ActionIgnoreFlag, submissionID, "submission", "Restored: "+req.Reason)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Submission restored"})
}

// AdminUpdateUser handles PUT /admin/users/:id
func AdminUpdateUser(c *gin.Context) {
	targetUserID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Name       string `json:"name"`
		Username   string `json:"username"`
		Bio        string `json:"bio"`
		Email      string `json:"email"`
		Role       string `json:"role"`
		TrustScore int    `json:"trustScore"`
		IsBlocked  bool   `json:"isBlocked"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "id = ?", targetUserID).Error; err != nil {
			return err
		}

		// Check username uniqueness if changed
		if req.Username != "" && req.Username != user.Username {
			var count int64
			tx.Model(&models.User{}).Where("username = ? AND id != ?", req.Username, targetUserID).Count(&count)
			if count > 0 {
				return gorm.ErrInvalidData // Custom error would be better
			}
		}

		updates := map[string]interface{}{
			"name":        req.Name,
			"username":    req.Username,
			"bio":         req.Bio,
			"email":       req.Email,
			"role":        models.Role(req.Role),
			"trust_score": req.TrustScore,
			"is_blocked":  req.IsBlocked,
		}

		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}

		return logAdminAction(tx, adminID, models.ActionUpdateUser, targetUserID, "user", "Updated by Admin")
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// AdminDeleteUser handles DELETE /admin/users/:id
func AdminDeleteUser(c *gin.Context) {
	targetUserID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "id = ?", targetUserID).Error; err != nil {
			return err
		}

		// Soft delete or Hard delete? Usually soft delete is safer.
		// models.User has gorm.DeletedAt, so tx.Delete will soft delete.
		if err := tx.Delete(&user).Error; err != nil {
			return err
		}

		return logAdminAction(tx, adminID, models.ActionDeleteUser, targetUserID, "user", "Deleted by Admin")
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// AdminGetRolePermissions handles GET /admin/roles/permissions
func AdminGetRolePermissions(c *gin.Context) {
	var perms []models.RolePermission
	database.DB.Find(&perms)
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

// AdminUpdateRolePermission handles PUT /admin/roles/permissions
func AdminUpdateRolePermission(c *gin.Context) {
	adminID := getAdminID(c)

	var req models.RolePermission
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role is required"})
		return
	}

	req.UpdatedAt = time.Now()
	req.UpdatedBy = adminID

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&req).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionUpdatePermissions, string(req.Role), "role", "Permissions updated for "+string(req.Role))
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated successfully", "permission": req})
}

// ============================================
// SYSTEM CONTROLS
// ============================================

// AdminGetSystemSettings returns all system settings
func AdminGetSystemSettings(c *gin.Context) {
	var settings []models.SystemSettings
	database.DB.Find(&settings)

	// Convert to map for easier frontend consumption
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	c.JSON(http.StatusOK, gin.H{"settings": settingsMap})
}

// AdminUpdateSystemSettings updates a system setting
func AdminUpdateSystemSettings(c *gin.Context) {
	adminID := getAdminID(c)

	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate key
	validKeys := map[string]bool{
		models.SettingMaintenanceMode:             true,
		models.SettingSubmissionsEnabled:          true,
		models.SettingSnippetsEnabled:             true,
		models.SettingContestsEnabled:             true,
		models.SettingRegistrationOpen:            true,
		models.SettingMaintenanceETA:              true,
		models.SettingFeatureSidebarXPStore:       true,
		models.SettingFeatureSidebarTrophyRoom:    true,
		models.SettingFeatureSidebarPractice:      true,
		models.SettingFeatureSidebarFeedback:      true,
		models.SettingFeatureSidebarRoadmaps:      true,
		models.SettingFeatureSidebarCommunity:     true,
		models.SettingFeatureInterfaceEngine:      true,
		models.SettingFeatureQuestsEnabled:        true,
		models.SettingFeatureSidebarLeaderboard:   true,
		models.SettingFeatureNotificationsEnabled: true,
		models.SettingFeatureSidebarNewBadge:      true,
		models.SettingSidebarBadges:               true,
		models.SettingBannerVisible:               true,
		models.SettingBannerTitle:                 true,
		models.SettingBannerBadge:                 true,
		models.SettingBannerContent:               true,
		models.SettingBannerLink:                  true,
	}
	if !validKeys[req.Key] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid setting key"})
		return
	}

	setting := models.SystemSettings{
		Key:       req.Key,
		Value:     req.Value,
		UpdatedBy: adminID,
		UpdatedAt: time.Now(),
	}

	// Upsert
	err := database.DB.Where("key = ?", req.Key).Assign(setting).FirstOrCreate(&setting).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, req.Key, "system", "Changed to: "+req.Value)

	c.JSON(http.StatusOK, gin.H{"message": "Setting updated", "setting": setting})
}

// PublicGetSystemStatus returns non-sensitive system info for maintenance pages
func PublicGetSystemStatus(c *gin.Context) {
	var settings []models.SystemSettings
	database.DB.Where("key IN ?", []string{
		models.SettingMaintenanceMode,
		models.SettingMaintenanceETA,
		models.SettingFeatureSidebarXPStore,
		models.SettingFeatureSidebarTrophyRoom,
		models.SettingFeatureSidebarPractice,
		models.SettingFeatureSidebarFeedback,
		models.SettingFeatureSidebarRoadmaps,
		models.SettingFeatureSidebarCommunity,
		models.SettingFeatureInterfaceEngine,
		models.SettingSidebarBadges,
		models.SettingBannerVisible,
		models.SettingBannerTitle,
		models.SettingBannerBadge,
		models.SettingBannerContent,
		models.SettingBannerLink,
	}).Find(&settings)

	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	c.JSON(http.StatusOK, gin.H{"settings": settingsMap})
}

// PublicGetLandingStats returns real platform stats for the landing page
func PublicGetLandingStats(c *gin.Context) {
	var stats struct {
		TotalUsers       int64          `json:"totalUsers"`
		TotalSubmissions int64          `json:"totalSubmissions"`
		TotalSnippets    int64          `json:"totalSnippets"`
		TotalContests    int64          `json:"totalContests"`
		UpcomingEvents   []models.Event `json:"upcomingEvents"`
		TopContestants   []models.User  `json:"topContestants"`
	}

	// 1. Basic Counts
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)
	database.DB.Model(&models.Submission{}).Count(&stats.TotalSubmissions)
	database.DB.Model(&models.Snippet{}).Count(&stats.TotalSnippets)
	database.DB.Model(&models.Event{}).Count(&stats.TotalContests)

	// 2. Upcoming Events (Next 3)
	database.DB.Where("status = ? AND start_time > ?", models.EventStatusUpcoming, time.Now()).
		Order("start_time asc").
		Limit(3).
		Find(&stats.UpcomingEvents)

	// 3. Top Contestants (by trust score or snippet count for now)
	database.DB.Where("onboarding_completed = ?", true).
		Order("trust_score desc, snippet_count desc").
		Limit(3).
		Find(&stats.TopContestants)

	c.JSON(http.StatusOK, stats)
}

// AdminTriggerRedeploy executes the redeployment script on the server
func AdminTriggerRedeploy(c *gin.Context) {
	adminID := getAdminID(c)

	var req struct {
		Mode string `json:"mode"` // "backend", "frontend", "all"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to 'all' if empty, or error? Let's default to all if valid json but empty field, or error if completely weird.
		// Actually BindJSON requires body.
		req.Mode = "all"
	}
	if req.Mode == "" {
		req.Mode = "all"
	}

	// Double check mode safety
	if req.Mode != "backend" && req.Mode != "frontend" && req.Mode != "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mode. Use 'backend', 'frontend', or 'all'"})
		return
	}

	// Log action before triggering (in case server restarts and we lose log context in memory, but DB is fine)
	logAdminAction(database.DB, adminID, models.ActionManageSystem, "system", "system", "Triggered Redeploy: "+req.Mode)

	// Trigger async to allow response to be sent
	go func(mode string) {
		cmd := exec.Command("/bin/bash", "/var/www/codestudio/redeploy.sh", mode)
		// We could capture output to a log file or database if needed. For now just log to stdout (server logs)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("❌ Redeploy Failed: %v\nOutput: %s", err, string(output))
			// Ideally we would update a DB record saying "Deployment Failed"
		} else {
			log.Printf("✅ Redeploy Success (%s)\nOutput: %s", mode, string(output))
		}
	}(req.Mode)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Redeployment triggered for %s. Server may restart shortly.", req.Mode),
	})
}
