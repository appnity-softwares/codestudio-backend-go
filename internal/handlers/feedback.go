package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// CreateFeedback handles posting new feedback (Rate limited: 3/hr)
func CreateFeedback(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 1. Rate Limiting (1 message per 30 seconds)
	allowed, err := database.CheckRateLimit(userID, 1, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "You're sending messages too fast. Please wait 30 seconds."})
		return
	}

	// 2. Parse Input
	var input struct {
		Content  string                  `json:"content" binding:"required,max=500"`
		Category models.FeedbackCategory `json:"category"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Create Message
	feedback := models.FeedbackMessage{
		UserID:   userID,
		Content:  input.Content,
		Category: input.Category,
	}
	if feedback.Category == "" {
		feedback.Category = models.CategoryOther
	}

	if err := database.DB.Create(&feedback).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback"})
		return
	}

	// Preload user for response
	database.DB.Preload("User").First(&feedback, "id = ?", feedback.ID)

	// 4. Invalidate Cache (Latest feed)
	go database.CacheInvalidate("feedback:latest*")

	c.JSON(http.StatusCreated, feedback)
}

// GetFeedback returns feedback list (paginated, sorted)
func GetFeedback(c *gin.Context) {
	sort := c.DefaultQuery("sort", "latest")
	offset := c.DefaultQuery("offset", "0")
	cacheKey := "feedback:" + sort + ":offset:" + offset

	// Check Cache (only if first page)
	if offset == "0" {
		var cached []models.FeedbackMessage
		if err := database.CacheGet(cacheKey, &cached); err == nil {
			// Even with cache, we need to check "HasReacted" for the current user
			userID := c.GetString("userId")
			if userID != "" {
				checkReactionsForMessages(userID, cached)
			}
			c.JSON(http.StatusOK, gin.H{"data": cached, "source": "cache"})
			return
		}
	}

	var messages []models.FeedbackMessage
	query := database.DB.Preload("User").Model(&models.FeedbackMessage{})

	if sort == "top" {
		query = query.Order("upvotes DESC, created_at DESC")
	} else {
		query = query.Order("created_at DESC")
	}

	query.Limit(50).Offset(0) // Simplified

	if err := query.Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feedback"})
		return
	}

	// Check if current user reacted
	userID := c.GetString("userId")
	if userID != "" {
		checkReactionsForMessages(userID, messages)
	}

	// Set Cache (TTL 30s) - Store RAW messages (without HasReacted) ??
	// Ideally we shouldn't cache HasReacted state, but we are.
	// Since HasReacted is virtual `gorm:"-"`, it might NOT be serialized to JSON for Redis if we didn't add JSON tag?
	// It HAS json tag. So it WILL be cached.
	// This is a bug: If user A caches it, User B sees User A's reactions?
	// FIX: Reset HasReacted before caching!

	messagesForCache := make([]models.FeedbackMessage, len(messages))
	copy(messagesForCache, messages)
	for i := range messagesForCache {
		messagesForCache[i].HasReacted = false
	}

	if offset == "0" {
		go database.CacheSet(cacheKey, messagesForCache, 30*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{"data": messages, "source": "db"})
}

func checkReactionsForMessages(userID string, messages []models.FeedbackMessage) {
	if len(messages) == 0 {
		return
	}
	var reactionIDs []string
	database.DB.Model(&models.FeedbackReaction{}).
		Where("user_id = ?", userID).
		Pluck("message_id", &reactionIDs)

	reactionMap := make(map[string]bool)
	for _, id := range reactionIDs {
		reactionMap[id] = true
	}

	for i := range messages {
		if reactionMap[messages[i].ID] {
			messages[i].HasReacted = true
		} else {
			messages[i].HasReacted = false // Reset in case it came from dirty cache
		}
	}
}

// ReactFeedback handles toggling reactions (Upvote)
func ReactFeedback(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	messageID := c.Param("id")

	var reaction models.FeedbackReaction
	result := database.DB.Where("user_id = ? AND message_id = ?", userID, messageID).First(&reaction)

	tx := database.DB.Begin()

	if result.Error == gorm.ErrRecordNotFound {
		// Add Reaction
		newReaction := models.FeedbackReaction{UserID: userID, MessageID: messageID}
		if err := tx.Create(&newReaction).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to react"})
			return
		}
		if err := tx.Model(&models.FeedbackMessage{}).Where("id = ?", messageID).UpdateColumn("upvotes", gorm.Expr("upvotes + ?", 1)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update count"})
			return
		}
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"status": "added"})

	} else {
		// Remove Reaction
		if err := tx.Delete(&reaction).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove reaction"})
			return
		}
		if err := tx.Model(&models.FeedbackMessage{}).Where("id = ?", messageID).UpdateColumn("upvotes", gorm.Expr("upvotes - ?", 1)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update count"})
			return
		}
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"status": "removed"})
	}

	go database.CacheInvalidate("feedback:*")
}
