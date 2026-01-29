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

	// Optimized query using DISTINCT ON for guaranteed latest message per partner
	// distinct on (partner_id) ... order by partner_id, created_at desc
	query := `
		SELECT DISTINCT ON (partner_id)
			u.id, COALESCE(u.username, ''), COALESCE(u.name, u.username, ''), COALESCE(u.image, ''),
			m.id as last_message_id, COALESCE(m.content, '') as last_message_content, m.created_at as last_message_at, m.sender_id as last_message_sender_id,
			(SELECT count(*) FROM messages WHERE sender_id = u.id AND recipient_id = ? AND is_read = false) as unread_count
		FROM messages m,
		LATERAL (
			SELECT CASE WHEN sender_id = ? THEN recipient_id ELSE sender_id END as partner_id
		) p
		JOIN "User" u ON u.id = p.partner_id
		WHERE m.sender_id = ? OR m.recipient_id = ?
		ORDER BY partner_id, m.created_at DESC
	`

	rows, err := database.DB.Raw(query, userId, userId, userId, userId).Rows()
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

// SendMessage handles sending a text message with production-grade security
func SendMessage(c *gin.Context) {
	senderID := c.MustGet("userId").(string)
	var req struct {
		RecipientID     string `json:"recipientId" binding:"required"`
		Content         string `json:"content" binding:"required"`
		Type            string `json:"type"`            // text, code, image, system
		ClientMessageID string `json:"clientMessageId"` // For deduplication
		ReplyToID       string `json:"replyToId"`       // For threading
		Metadata        string `json:"metadata"`        // JSON metadata (code language, etc.)
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 1. Set default type
	if req.Type == "" {
		req.Type = "text"
	}

	// 2. Validate message type
	if !ValidateMessageType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message type. Must be: text, code, image, or system"})
		return
	}

	// 3. Block system messages from user API (system messages are server-generated only)
	if req.Type == "system" {
		c.JSON(http.StatusForbidden, gin.H{"error": "System messages cannot be sent via API"})
		return
	}

	// 4. SECURITY: Validate/Sanitize content based on type
	var sanitizedContent string
	if req.Type == "image" {
		// Image messages: Validate URL instead of sanitizing HTML
		if err := ValidateImageURL(req.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sanitizedContent = req.Content // URL is safe if validation passed
	} else {
		// Text/Code messages: Sanitize to prevent XSS
		var err error
		sanitizedContent, err = SanitizeMessageContent(req.Content, req.Type)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// 5. DEDUPLICATION: Check if message with this ClientMessageID already exists
	if req.ClientMessageID != "" {
		var existing models.Message
		if err := database.DB.Where("client_message_id = ?", req.ClientMessageID).First(&existing).Error; err == nil {
			// Already exists, return the existing message (idempotent)
			database.DB.Preload("Sender").Preload("Recipient").First(&existing, "id = ?", existing.ID)
			c.JSON(http.StatusOK, gin.H{"message": existing, "deduplicated": true})
			return
		}
	}

	// 6. Sanitize metadata if present
	metadata := ""
	if req.Metadata != "" {
		metadata = SanitizeCodeMetadata(req.Metadata)
	}

	// 7. Build message
	msg := models.Message{
		SenderID:    senderID,
		RecipientID: req.RecipientID,
		Content:     sanitizedContent,
		Type:        req.Type,
		Status:      "sent",
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}

	// Set ClientMessageID if provided (nullable)
	if req.ClientMessageID != "" {
		msg.ClientMessageID = &req.ClientMessageID
	}

	// 8. Handle reply threading
	if req.ReplyToID != "" {
		msg.ReplyToID = &req.ReplyToID
	}

	// 9. Persist to database
	if err := database.DB.Create(&msg).Error; err != nil {
		fmt.Printf("[Chat] SendMessage FAILED for sender %s to %s: %v\n", senderID, req.RecipientID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send message",
			"details": err.Error(),
		})
		return
	}

	// 9. Populate relations for return
	database.DB.Preload("Sender").Preload("Recipient").First(&msg, "id = ?", msg.ID)

	// 10. Real-time emission
	if SocketServer != nil {
		data := map[string]interface{}{
			"message": msg,
		}
		// Send to recipient
		SocketServer.BroadcastToRoom("/", msg.RecipientID, "receive_message", data)
		// Send to sender for multi-device sync
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

// ============================================
// PHASE 7: REACTIONS
// ============================================

// AddReaction adds or removes a reaction to a message (toggle behavior)
func AddReaction(c *gin.Context) {
	userID := c.MustGet("userId").(string)
	messageID := c.Param("messageId")

	var req struct {
		Emoji string `json:"emoji" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Emoji is required"})
		return
	}

	// Validate emoji is in allowed list
	if !models.IsValidReactionEmoji(req.Emoji) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid emoji",
			"allowed": models.AllowedReactionEmojis,
		})
		return
	}

	// Check message exists
	var msg models.Message
	if err := database.DB.First(&msg, "id = ?", messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// Toggle: Check if reaction already exists
	var existing models.MessageReaction
	err := database.DB.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, req.Emoji).First(&existing).Error

	if err == nil {
		// Reaction exists - remove it (toggle off)
		database.DB.Delete(&existing)

		// Broadcast removal
		if SocketServer != nil {
			SocketServer.BroadcastToRoom("/", msg.SenderID, "reaction_removed", map[string]interface{}{
				"messageId":  messageID,
				"userId":     userID,
				"emoji":      req.Emoji,
				"reactionId": existing.ID,
			})
			if msg.RecipientID != msg.SenderID {
				SocketServer.BroadcastToRoom("/", msg.RecipientID, "reaction_removed", map[string]interface{}{
					"messageId":  messageID,
					"userId":     userID,
					"emoji":      req.Emoji,
					"reactionId": existing.ID,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{"removed": true, "emoji": req.Emoji})
		return
	}

	// Create new reaction
	reaction := models.MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     req.Emoji,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&reaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reaction"})
		return
	}

	// Load user for response
	database.DB.Preload("User").First(&reaction, "id = ?", reaction.ID)

	// Broadcast to conversation participants
	if SocketServer != nil {
		reactionData := map[string]interface{}{
			"reaction": reaction,
		}
		SocketServer.BroadcastToRoom("/", msg.SenderID, "reaction_added", reactionData)
		if msg.RecipientID != msg.SenderID {
			SocketServer.BroadcastToRoom("/", msg.RecipientID, "reaction_added", reactionData)
		}
	}

	c.JSON(http.StatusOK, gin.H{"reaction": reaction, "added": true})
}

// GetReactions returns all reactions for a message
func GetReactions(c *gin.Context) {
	messageID := c.Param("messageId")

	var reactions []models.MessageReaction
	if err := database.DB.Preload("User").Where("message_id = ?", messageID).Find(&reactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reactions"})
		return
	}

	// Group by emoji for easy display
	grouped := make(map[string][]models.MessageReaction)
	for _, r := range reactions {
		grouped[r.Emoji] = append(grouped[r.Emoji], r)
	}

	c.JSON(http.StatusOK, gin.H{
		"reactions": reactions,
		"grouped":   grouped,
		"count":     len(reactions),
	})
}

// GetTotalUnreadMessages GET /chat/unread/total
func GetTotalUnreadMessages(c *gin.Context) {
	userID := c.MustGet("userId").(string)

	var count int64
	// Count unread messages where I am the recipient
	database.DB.Model(&models.Message{}).
		Where("recipient_id = ? AND is_read = ?", userID, false).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"count": count})
}
