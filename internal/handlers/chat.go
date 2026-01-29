package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// GetContacts returns users that the current user is following (Linked)
func GetContacts(c *gin.Context) {
	userId := c.MustGet("userId").(string)

	var contacts []models.User
	// Find users that I follow (LinkerID = me)
	// We might also want users who follow me (LinkedID = me) i.e. Friends/Mutuals?
	// For now, let's just return users I follow as potential contacts.
	err := database.DB.Table("\"User\"").
		Joins("JOIN \"UserLink\" ON \"UserLink\".linked_id = \"User\".id").
		Where("\"UserLink\".linker_id = ?", userId).
		Find(&contacts).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contacts": contacts})
}

// GetConversations returns a list of active conversations (by recent messages)
func GetConversations(c *gin.Context) {
	userId := c.MustGet("userId").(string)

	// Optimized query using latest message timestamp for better precision
	// Handles both sender and recipient roles for the current user
	query := `
		WITH PartnerLatest AS (
			SELECT 
				CASE WHEN sender_id = ? THEN recipient_id ELSE sender_id END as partner_id,
				MAX(created_at) as last_msg_at
			FROM messages
			WHERE sender_id = ? OR recipient_id = ?
			GROUP BY 1
		)
		SELECT 
			u.id, COALESCE(u.username, ''), COALESCE(u.name, u.username, ''), COALESCE(u.image, ''),
			m.id as last_message_id, COALESCE(m.content, '') as last_message_content, m.created_at as last_message_at, m.sender_id as last_message_sender_id,
			(SELECT count(*) FROM messages WHERE sender_id = u.id AND recipient_id = ? AND is_read = false) as unread_count
		FROM PartnerLatest pt
		JOIN "User" u ON u.id = pt.partner_id
		JOIN messages m ON (
			(m.sender_id = u.id AND m.recipient_id = ?) OR 
			(m.sender_id = ? AND m.recipient_id = u.id)
		) AND m.created_at = pt.last_msg_at
		ORDER BY pt.last_msg_at DESC
	`

	rows, err := database.DB.Raw(query, userId, userId, userId, userId, userId, userId).Rows()
	if err != nil {
		fmt.Printf("Error fetching optimized conversations: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch conversations"})
		return
	}
	defer rows.Close()

	var conversations []map[string]interface{}
	for rows.Next() {
		var u models.User
		var lastMsg models.Message
		var unread int64

		var lastMsgID, lastMsgContent, lastMsgSenderID string
		var lastMsgAt time.Time

		err := rows.Scan(
			&u.ID, &u.Username, &u.Name, &u.Image,
			&lastMsgID, &lastMsgContent, &lastMsgAt, &lastMsgSenderID,
			&unread,
		)
		if err != nil {
			fmt.Printf("Scan error in GetConversations: %v\n", err)
			continue
		}

		lastMsg.ID = lastMsgID
		lastMsg.Content = lastMsgContent
		lastMsg.CreatedAt = lastMsgAt
		lastMsg.SenderID = lastMsgSenderID

		conversations = append(conversations, map[string]interface{}{
			"user":        u,
			"lastMessage": lastMsg,
			"unreadCount": unread,
		})
	}

	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

// GetMessages returns messages for a specific user (DM)
func GetMessages(c *gin.Context) {
	currentUserID := c.MustGet("userId").(string)
	otherUserID := c.Query("userId")

	if otherUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}

	var messages []models.Message
	err := database.DB.Where(
		"(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		currentUserID, otherUserID, otherUserID, currentUserID,
	).Order("created_at asc").Preload("Sender").Preload("Recipient").Find(&messages).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendMessage handles sending a text message
func SendMessage(c *gin.Context) {
	senderID := c.MustGet("userId").(string)
	var req struct {
		RecipientID string `json:"recipientId" binding:"required"`
		Content     string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	msg := models.Message{
		SenderID:    senderID,
		RecipientID: req.RecipientID,
		Content:     req.Content,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send message",
			"details": err.Error(),
		})
		return
	}

	// Populate relations for return
	database.DB.Preload("Sender").Preload("Recipient").First(&msg, "id = ?", msg.ID)

	// Real-time emission
	if SocketServer != nil {
		data := map[string]interface{}{
			"message": msg,
		}
		// Send to recipient
		SocketServer.BroadcastToRoom("/", msg.RecipientID, "receive_message", data)
		// Optionally send to sender for multi-device sync
		SocketServer.BroadcastToRoom("/", msg.SenderID, "receive_message", data)
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// MarkRead marks messages from a sender as read
func MarkRead(c *gin.Context) {
	currentUserID := c.MustGet("userId").(string)
	senderID := c.Param("senderId")

	result := database.DB.Model(&models.Message{}).
		Where("sender_id = ? AND recipient_id = ? AND is_read = ?", senderID, currentUserID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark read"})
		return
	}

	// Notify sender that messages have been read
	if SocketServer != nil && result.RowsAffected > 0 {
		SocketServer.BroadcastToRoom("/", senderID, "message_read", map[string]interface{}{
			"senderId": currentUserID, // The one who read the messages
		})
	}

	c.JSON(http.StatusOK, gin.H{"markedRead": result.RowsAffected})
}
