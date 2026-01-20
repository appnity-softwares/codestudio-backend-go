package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// ============================================
// FEEDBACK MODERATION (Admin)
// ============================================

// AdminUpdateFeedbackStatus changes the lifecycle status of feedback
func AdminUpdateFeedbackStatus(c *gin.Context) {
	feedbackID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		string(models.StatusOpen):      true,
		string(models.StatusReviewing): true,
		string(models.StatusPlanned):   true,
		string(models.StatusShipped):   true,
		string(models.StatusClosed):    true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Valid: OPEN, REVIEWING, PLANNED, SHIPPED, CLOSED"})
		return
	}

	var feedback models.FeedbackMessage
	if err := database.DB.First(&feedback, "id = ?", feedbackID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feedback not found"})
		return
	}

	updates := map[string]interface{}{
		"status": req.Status,
	}

	// Auto-lock on Shipped or Closed
	if req.Status == string(models.StatusShipped) || req.Status == string(models.StatusClosed) {
		updates["is_locked"] = true
	}

	if err := database.DB.Model(&feedback).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	// Invalidate cache
	go database.CacheInvalidate("feedback:*")

	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, feedbackID, "feedback", "Status changed to "+req.Status)

	c.JSON(http.StatusOK, gin.H{"message": "Status updated", "status": req.Status})
}

// AdminLockFeedback locks or unlocks voting/replies on feedback
func AdminLockFeedback(c *gin.Context) {
	feedbackID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Locked bool `json:"locked"`
	}
	c.BindJSON(&req)

	if err := database.DB.Model(&models.FeedbackMessage{}).
		Where("id = ?", feedbackID).
		Update("is_locked", req.Locked).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lock status"})
		return
	}

	go database.CacheInvalidate("feedback:*")

	action := "unlocked"
	if req.Locked {
		action = "locked"
	}
	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, feedbackID, "feedback", "Feedback "+action)

	c.JSON(http.StatusOK, gin.H{"message": "Feedback " + action, "locked": req.Locked})
}

// AdminHideFeedback hides or unhides feedback from public view
func AdminHideFeedback(c *gin.Context) {
	feedbackID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Hidden bool `json:"hidden"`
	}
	c.BindJSON(&req)

	if err := database.DB.Model(&models.FeedbackMessage{}).
		Where("id = ?", feedbackID).
		Update("is_hidden", req.Hidden).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update visibility"})
		return
	}

	go database.CacheInvalidate("feedback:*")

	action := "unhidden"
	if req.Hidden {
		action = "hidden"
	}
	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, feedbackID, "feedback", "Feedback "+action)

	c.JSON(http.StatusOK, gin.H{"message": "Feedback " + action, "hidden": req.Hidden})
}

// AdminPinFeedback pins or unpins feedback to the top
func AdminPinFeedback(c *gin.Context) {
	feedbackID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Pinned bool `json:"pinned"`
	}
	c.BindJSON(&req)

	if err := database.DB.Model(&models.FeedbackMessage{}).
		Where("id = ?", feedbackID).
		Update("is_pinned", req.Pinned).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pin status"})
		return
	}

	go database.CacheInvalidate("feedback:*")

	action := "unpinned"
	if req.Pinned {
		action = "pinned"
	}
	logAdminAction(database.DB, adminID, models.ActionPinSnippet, feedbackID, "feedback", "Feedback "+action)

	c.JSON(http.StatusOK, gin.H{"message": "Feedback " + action, "pinned": req.Pinned})
}

// AdminConvertToChangelog links feedback to a changelog entry
func AdminConvertToChangelog(c *gin.Context) {
	feedbackID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		ChangelogID string `json:"changelogId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"changelog_id": req.ChangelogID,
		"status":       models.StatusShipped,
		"is_locked":    true,
	}

	if err := database.DB.Model(&models.FeedbackMessage{}).
		Where("id = ?", feedbackID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link"})
		return
	}

	go database.CacheInvalidate("feedback:*")

	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, feedbackID, "feedback", "Linked to changelog: "+req.ChangelogID)

	c.JSON(http.StatusOK, gin.H{"message": "Feedback linked to changelog and marked as shipped"})
}

// AdminListFeedback returns all feedback for admin moderation
func AdminListFeedback(c *gin.Context) {
	var feedback []models.FeedbackMessage

	query := database.DB.Preload("User").Order("is_pinned DESC, created_at DESC")

	// Include hidden for admins
	if err := query.Find(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feedback": feedback, "count": len(feedback)})
}
