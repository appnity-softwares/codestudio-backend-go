package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// GetNotifications GET /notifications
func GetNotifications(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var notifications []models.Notification
	if err := database.DB.Preload("Actor").Preload("Snippet").Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

// GetUnreadCount GET /notifications/unread-count
func GetUnreadCount(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var count int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetAggregateUnreadCount GET /notifications/aggregate
func GetAggregateUnreadCount(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var notificationCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&notificationCount)

	var messageCount int64
	database.DB.Model(&models.Message{}).Where("recipient_id = ? AND is_read = ?", userID, false).Count(&messageCount)

	var requestCount int64
	database.DB.Model(&models.LinkRequest{}).Where("receiver_id = ? AND status = ?", userID, models.LinkRequestPending).Count(&requestCount)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notificationCount,
		"messages":      messageCount,
		"linkRequests":  requestCount,
		"total":         notificationCount + messageCount + requestCount,
	})
}

// MarkNotificationRead PUT /notifications/:id/read
func MarkNotificationRead(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	notificationID := c.Param("id")

	var notification models.Notification
	if err := database.DB.First(&notification, "id = ?", notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	if notification.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	notification.IsRead = true
	database.DB.Save(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

// MarkAllNotificationsRead PUT /notifications/read-all
func MarkAllNotificationsRead(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true)

	c.JSON(http.StatusOK, gin.H{"message": "All marked as read"})
}

// DeleteNotification DELETE /notifications/:id
func DeleteNotification(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	notificationID := c.Param("id")

	var notification models.Notification
	if err := database.DB.First(&notification, "id = ?", notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	if notification.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	database.DB.Delete(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// CreateNotification helper to persist and send real-time notification
func CreateNotification(tx *gorm.DB, notification models.Notification) error {
	if err := tx.Create(&notification).Error; err != nil {
		fmt.Printf("Error creating notification: %v\n", err)
		return err
	}

	// Load Actor and Snippet for the frontend
	var fullNotification models.Notification
	// We must use 'tx' here because the row is not committed yet if inside a transaction
	if err := tx.Preload("Actor").Preload("Snippet").First(&fullNotification, "id = ?", notification.ID).Error; err != nil {
		// Log error but don't fail the transaction, just skip real-time
		// fmt.Printf("Error fetching notification for real-time: %v\n", err)
	}

	data := map[string]interface{}{
		"id":        fullNotification.ID,
		"type":      fullNotification.Type,
		"message":   fullNotification.Message,
		"actor":     fullNotification.Actor,
		"snippet":   fullNotification.Snippet,
		"createdAt": fullNotification.CreatedAt,
		"isRead":    fullNotification.IsRead,
	}

	SendNotificationToUser(notification.UserID, data)
	return nil
}

// NotifyNewBadges sends notifications for a list of earned badges
func NotifyNewBadges(userID string, badges []models.Badge) {
	for _, b := range badges {
		notification := models.Notification{
			UserID:  userID,
			Type:    models.NotificationTypeAchievement,
			Message: "Unlocked Badge: " + b.Name,
		}
		CreateNotification(database.DB, notification)
	}
}
