package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// GetChatHistory returns messages between current user and target user
func GetChatHistory(c *gin.Context) {
	currentUserID, _ := c.Get("userId")
	targetUserID := c.Query("userId")

	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user ID required"})
		return
	}

	var messages []models.Message
	if err := database.DB.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		currentUserID, targetUserID, targetUserID, currentUserID,
	).Order("created_at asc").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// MarkMessagesAsRead marks all messages from a sender to current user as read
func MarkMessagesAsRead(c *gin.Context) {
	currentUserID, _ := c.Get("userId")
	senderID := c.Param("senderId")

	if senderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sender ID required"})
		return
	}

	// Update all unread messages from sender to current user
	result := database.DB.Model(&models.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND is_read = ?", senderID, currentUserID, false).
		Update("is_read", true)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark messages as read"})
		return
	}

	// Notify sender via socket that their messages were read
	if SocketServer != nil {
		chatId := sortedChatId(currentUserID.(string), senderID)
		SocketServer.BroadcastToRoom("/", chatId, "messages_read", gin.H{
			"readBy":   currentUserID,
			"senderId": senderID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"markedRead": result.RowsAffected})
}

// Helper to create consistent chat ID
func sortedChatId(a, b string) string {
	if a < b {
		return a + "-" + b
	}
	return b + "-" + a
}

// ListChatContacts returns unique users the current user has chatted with, sorted by most recent message
func ListChatContacts(c *gin.Context) {
	currentUserID, _ := c.Get("userId")

	contacts := []models.User{}
	// Complex query to get users sorted by max message time
	// We join User with Messages to find the latest message timestamp for each contact relative to current user
	err := database.DB.Raw(`
		SELECT u.* 
		FROM "User" u
		JOIN (
			SELECT 
				CASE 
					WHEN sender_id = ? THEN receiver_id 
					ELSE sender_id 
				END as contact_id,
				MAX(created_at) as last_msg_time
			FROM messages
			WHERE sender_id = ? OR receiver_id = ?
			GROUP BY contact_id
		) m ON u.id = m.contact_id
		ORDER BY m.last_msg_time DESC
	`, currentUserID, currentUserID, currentUserID).Scan(&contacts).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contacts": contacts})
}
