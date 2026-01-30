package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// GetActivityFeed returns the global activity feed for the platform
func GetActivityFeed(c *gin.Context) {
	userId, exists := c.Get("userId")

	query := database.DB.Model(&models.UserActivity{}).Preload("Actor")

	// Filter by type
	activityType := c.Query("type")
	if activityType != "" {
		query = query.Where("type = ?", activityType)
	}

	// Filter by following (Requires Auth)
	followingOnly := c.Query("following") == "true"
	if followingOnly && exists {
		var followedIds []string
		database.DB.Table("\"UserLink\"").Where("linker_id = ? AND \"deletedAt\" IS NULL", userId).Pluck("linked_id", &followedIds)

		// If following nobody, show nothing or maybe handle gracefully
		if len(followedIds) > 0 {
			query = query.Where("actor_id IN ?", followedIds)
		} else {
			// User is not following anyone
			c.JSON(http.StatusOK, gin.H{"activities": []models.UserActivity{}})
			return
		}
	}

	var activities []models.UserActivity
	if err := query.Order("created_at desc").Limit(50).Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}
