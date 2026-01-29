package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// LinkUser handles POST /users/:id/link (Follow)
func LinkUser(c *gin.Context) {
	fmt.Printf("[Social] Processing Link request for target: %s\n", c.Param("username"))
	// 1. Get Auth User
	linkerID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Get Target User
	targetID := c.Param("username")
	if linkerID.(string) == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot link yourself"})
		return
	}

	var targetUser models.User
	// Search by username or ID
	if err := database.DB.Where("username = ? OR id = ?", targetID, targetID).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	// Use the actual ID from the found user
	actualTargetID := targetUser.ID

	if linkerID.(string) == actualTargetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot link yourself"})
		return
	}

	// Blocking Check
	var blockCount int64
	database.DB.Model(&models.UserBlock{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
			linkerID, actualTargetID, actualTargetID, linkerID).
		Count(&blockCount)

	if blockCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot link with this user"})
		return
	}

	// 3. Handle Private Account Path
	if targetUser.Visibility == models.VisibilityPrivate {
		var existingReq models.LinkRequest
		if err := database.DB.Where("sender_id = ? AND receiver_id = ? AND status = ?", linkerID, actualTargetID, models.LinkRequestPending).First(&existingReq).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{"message": "Request already pending", "requested": true})
			return
		}

		newReq := models.LinkRequest{
			SenderID:   linkerID.(string),
			ReceiverID: actualTargetID,
			Status:     models.LinkRequestPending,
		}

		if err := database.DB.Create(&newReq).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to create link request",
				"details": err.Error(),
			})
			return
		}

		// Notify Target
		notification := models.Notification{
			UserID:  actualTargetID,
			ActorID: linkerID.(string),
			Type:    models.NotificationTypeLinkRequest,
			Message: "requested to link with you",
		}
		CreateNotification(database.DB, notification)

		c.JSON(http.StatusOK, gin.H{"message": "Link request sent", "requested": true})
		return
	}

	// 4. Public Path: Direct Link
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var existingLink models.UserLink
		// Check for existing link (including soft-deleted)
		err := tx.Unscoped().Where("linker_id = ? AND linked_id = ?", linkerID, actualTargetID).First(&existingLink).Error

		switch err {
		case nil:
			// Record found
			if existingLink.DeletedAt.Valid {
				// Soft-deleted -> Restore it
				if err := tx.Unscoped().Model(&existingLink).Update("deleted_at", nil).Error; err != nil {
					return fmt.Errorf("restore link: %w", err)
				}
				// Proceed to update counters below
			} else {
				// Already active -> Do nothing
				return nil
			}
		case gorm.ErrRecordNotFound:
			// No record -> Create new
			newLink := models.UserLink{
				LinkerID: linkerID.(string),
				LinkedID: actualTargetID,
			}
			if err := tx.Create(&newLink).Error; err != nil {
				return fmt.Errorf("create link: %w", err)
			}
		default:
			return err
		}

		// Merge all updates into a single call per user to minimize lock time
		// and use deterministic order to prevent deadlocks.
		users := []struct {
			id       string
			isTarget bool
		}{
			{id: actualTargetID, isTarget: true},
			{id: linkerID.(string), isTarget: false},
		}

		// Deterministic sort
		if users[0].id > users[1].id {
			users[0], users[1] = users[1], users[0]
		}

		// Track if we should award XP (only for NEW links)
		shouldAwardXP := err == gorm.ErrRecordNotFound

		for _, u := range users {
			updateData := map[string]interface{}{}

			// Only award XP if it's a fresh link (not a restore)
			if shouldAwardXP {
				updateData["xp"] = gorm.Expr("xp + 50")
			}

			if u.isTarget {
				updateData["linkersCount"] = gorm.Expr("\"linkersCount\" + 1")
			} else {
				updateData["linkedCount"] = gorm.Expr("\"linkedCount\" + 1")
			}

			if err := tx.Model(&models.User{}).Where("id = ?", u.id).Updates(updateData).Error; err != nil {
				return fmt.Errorf("update user %s: %w", u.id, err)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("[Social] Link processing failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link user", "details": err.Error()})
		return
	}

	// 5. Send Notification (Post-Transaction, Async)
	// Only send notification if it was a meaningful action (new or restore)
	// We'll send it regardless to inform user, but maybe change text?
	// Legacy behavior: Send it.
	go func(targetID, actorID string) {
		notification := models.Notification{
			UserID:  targetID,
			ActorID: actorID,
			Type:    models.NotificationTypeLinkAccept,
			Message: "linked with you (+50 Influence)",
		}
		if err := CreateNotification(database.DB, notification); err != nil {
			fmt.Printf("[Social] Notification async fail: %v\n", err)
		}
	}(actualTargetID, linkerID.(string))

	c.JSON(http.StatusOK, gin.H{"message": "User linked successfully", "linked": true})
	fmt.Printf("[Social] Link success: %s -> %s\n", linkerID, actualTargetID)
}

// AcceptLinkRequest POST /users/link-requests/:id/accept
func AcceptLinkRequest(c *gin.Context) {
	userID := c.MustGet("userId").(string)
	requestID := c.Param("id")

	var req models.LinkRequest
	if err := database.DB.First(&req, "id = ?", requestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	if req.ReceiverID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your request"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		req.Status = models.LinkRequestAccepted
		tx.Save(&req)

		// Create Link
		newLink := models.UserLink{
			LinkerID: req.SenderID,
			LinkedID: userID,
		}
		// Note: AcceptLinkRequest implies it wasn't a direct public link,
		// so likely a private profile. Less chance of spam farming here,
		// but ideally we should checking for history too.
		// For MVP/Safety, we assume Accept is a valid +XP event (strictly new).
		// But let's check duplication just in case.
		var existing models.UserLink
		if err := tx.Unscoped().Where("linker_id = ? AND linked_id = ?", req.SenderID, userID).First(&existing).Error; err == nil {
			// Exists
			if existing.DeletedAt.Valid {
				// Restore
				tx.Unscoped().Model(&existing).Update("deleted_at", nil)
				// Do NOT create new
			}
		} else {
			if err := tx.Create(&newLink).Error; err != nil {
				return err
			}
		}

		// Update counters
		// We'll just update counts, not XP here to keep logic simple/safe or give XP?
		// User specifically complained about the public Link/Unlink spam loop.
		// Let's give XP for Accepted Requests (it's hard to spam these as they require approval).
		// But to be consistent: "One time per user".
		// We should enforce it here too?
		// It's low risk. I'll stick to modifying LinkUser first as that is the spam vector.

		if err := tx.Model(&models.User{}).Where("id = ?", userID).UpdateColumn("linkersCount", gorm.Expr("COALESCE(\"linkersCount\", 0) + ?", 1)).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", req.SenderID).UpdateColumn("linkedCount", gorm.Expr("COALESCE(\"linkedCount\", 0) + ?", 1)).Error; err != nil {
			return err
		}

		// Note: AcceptLink doesn't seem to update XP in the original code?
		// Original code: Lines 210-213 only update counts.
		// So Link Request flow DOES NOT award XP?
		// Line 169 says "(+50 Influence)". That's for Direct Link.
		// If AcceptLink doesn't give XP, then no problem.

		// Notify Sender
		notification := models.Notification{
			UserID:  req.SenderID,
			ActorID: userID,
			Type:    models.NotificationTypeLinkAccept,
			Message: "accepted your link request",
		}
		return CreateNotification(tx, notification)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request accepted"})
}

// RejectLinkRequest POST /users/link-requests/:id/reject
func RejectLinkRequest(c *gin.Context) {
	userID := c.MustGet("userId").(string)
	requestID := c.Param("id")

	var req models.LinkRequest
	if err := database.DB.First(&req, "id = ?", requestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	if req.ReceiverID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your request"})
		return
	}

	req.Status = models.LinkRequestRejected
	database.DB.Save(&req)

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected"})
}

// ListLinkRequests GET /users/link-requests
func ListLinkRequests(c *gin.Context) {
	userID := c.MustGet("userId").(string)

	var requests []models.LinkRequest
	if err := database.DB.Preload("Sender").Where("receiver_id = ? AND status = ?", userID, models.LinkRequestPending).Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// UnlinkUser handles DELETE /users/:id/link (Unfollow)
func UnlinkUser(c *gin.Context) {
	linkerID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	targetInput := c.Param("username")

	var targetUser models.User
	if err := database.DB.Where("username = ? OR id = ?", targetInput, targetInput).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	actualTargetID := targetUser.ID

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Check for Pending Request (Cancel it)
		var pendingReq models.LinkRequest
		if err := tx.Where("sender_id = ? AND receiver_id = ? AND status = ?", linkerID, actualTargetID, models.LinkRequestPending).First(&pendingReq).Error; err == nil {
			// Found pending request -> Delete it
			if err := tx.Delete(&pendingReq).Error; err != nil {
				return err
			}
			return nil // Done
		}

		// 2. Check for Active Link (Unfollow)
		var link models.UserLink
		if err := tx.Model(&models.UserLink{}).Where("linker_id = ? AND linked_id = ?", linkerID, actualTargetID).First(&link).Error; err != nil {
			return nil // Idempotent (Nothing to unlink)
		}

		// Use Hard Delete to keep table clean and avoid unique index issues on re-linking
		if err := tx.Unscoped().Delete(&link).Error; err != nil {
			return err
		}

		// Deterministic locking order
		users := []struct {
			id       string
			isTarget bool
		}{
			{id: actualTargetID, isTarget: true},
			{id: linkerID.(string), isTarget: false},
		}
		if users[0].id > users[1].id {
			users[0], users[1] = users[1], users[0]
		}

		for _, u := range users {
			// REMOVED XP DEDUCTION here to support "Permanent One-Time" Influence
			updateData := map[string]interface{}{} // "xp": gorm.Expr("GREATEST(xp - 50, 0)")

			if u.isTarget {
				updateData["linkersCount"] = gorm.Expr("GREATEST(\"linkersCount\" - 1, 0)")
			} else {
				updateData["linkedCount"] = gorm.Expr("GREATEST(\"linkedCount\" - 1, 0)")
			}

			if err := tx.Model(&models.User{}).Where("id = ?", u.id).Updates(updateData).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("[Social] Unlink failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unlinked successfully", "linked": false})
	fmt.Printf("[Social] Unlink success: %s -> %s\n", linkerID, actualTargetID)
}

// GetLinkers handles GET /users/:id/linkers (Followers)
func GetLinkers(c *gin.Context) {
	targetInput := c.Param("username")
	var targetUser models.User
	if err := database.DB.Where("username = ? OR id = ?", targetInput, targetInput).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var links []models.UserLink
	if err := database.DB.Model(&models.UserLink{}).Preload("Linker").Where("linked_id = ?", targetUser.ID).Limit(50).Order("created_at desc").Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch linkers"})
		return
	}

	users := make([]gin.H, 0)
	for _, l := range links {
		users = append(users, gin.H{
			"id":       l.Linker.ID,
			"username": l.Linker.Username,
			"name":     l.Linker.Name,
			"image":    l.Linker.Image,
			"bio":      l.Linker.Bio,
			"linkedAt": l.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"linkers": users})
}

// GetLinked handles GET /users/:id/linked (Following)
func GetLinked(c *gin.Context) {
	targetInput := c.Param("username")
	var targetUser models.User
	if err := database.DB.Where("username = ? OR id = ?", targetInput, targetInput).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var links []models.UserLink
	if err := database.DB.Model(&models.UserLink{}).Preload("Linked").Where("linker_id = ?", targetUser.ID).Limit(50).Order("created_at desc").Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch linked users"})
		return
	}

	users := make([]gin.H, 0)
	for _, l := range links {
		users = append(users, gin.H{
			"id":       l.Linked.ID,
			"username": l.Linked.Username,
			"name":     l.Linked.Name,
			"image":    l.Linked.Image,
			"bio":      l.Linked.Bio,
			"linkedAt": l.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"linked": users})
}

// CheckLinkStatus handles GET /users/:id/link/status
func CheckLinkStatus(c *gin.Context) {
	linkerID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	targetInput := c.Param("username")

	var targetUser models.User
	if err := database.DB.Where("username = ? OR id = ?", targetInput, targetInput).Select("id").First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var count int64
	database.DB.Model(&models.UserLink{}).Where("linker_id = ? AND linked_id = ?", linkerID, targetUser.ID).Count(&count)

	if count > 0 {
		c.JSON(http.StatusOK, gin.H{"linked": true, "status": "LINKED"})
		return
	}

	// Check for pending request
	var reqCount int64
	database.DB.Model(&models.LinkRequest{}).Where("sender_id = ? AND receiver_id = ? AND status = ?", linkerID, targetUser.ID, models.LinkRequestPending).Count(&reqCount)

	if reqCount > 0 {
		c.JSON(http.StatusOK, gin.H{"linked": false, "status": "PENDING"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"linked": false, "status": "NONE"})
}

// --- Snippet Engagement ---

// ToggleLikeSnippet handles POST /snippets/:id/like
func ToggleLikeSnippet(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	snippetID := c.Param("id")

	// Transaction to toggle like
	var liked bool
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var like models.SnippetLike
		result := tx.Where("user_id = ? AND snippet_id = ?", userID, snippetID).First(&like)

		if result.Error == nil {
			// Like exists -> Unlike
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			// Decrement counter (but not below 0)
			if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error; err != nil {
				return err
			}
			liked = false
		} else {
			// Like doesn't exist -> Like

			// 1. Remove Dislike if exists (Mutually Exclusive)
			var dislike models.SnippetDislike
			if err := tx.Where("user_id = ? AND snippet_id = ?", userID, snippetID).First(&dislike).Error; err == nil {
				if err := tx.Delete(&dislike).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).UpdateColumn("dislikes_count", gorm.Expr("GREATEST(dislikes_count - 1, 0)")).Error; err != nil {
					return err
				}
			}

			// 2. Create Like
			newLike := models.SnippetLike{
				UserID:    userID.(string),
				SnippetID: snippetID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			// Increment counter
			if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).UpdateColumn("likes_count", gorm.Expr("likes_count + ?", 1)).Error; err != nil {
				return err
			}

			// Create Notification
			var snippet models.Snippet
			if err := tx.Select("\"authorId\"", "title").First(&snippet, "id = ?", snippetID).Error; err == nil {
				if snippet.AuthorID != userID.(string) {
					notification := models.Notification{
						UserID:    snippet.AuthorID,
						ActorID:   userID.(string),
						Type:      models.NotificationTypeLike,
						SnippetID: &snippetID,
						Message:   "liked your snippet: " + snippet.Title,
					}
					CreateNotification(tx, notification)
				}
			}

			liked = true
		}
		return nil
	})

	if err != nil {
		fmt.Printf("[Social] ToggleLikeSnippet FAILED for snippet %s, user %s: %v\n", snippetID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle like", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"liked": liked})
}

// ToggleDislikeSnippet handles POST /snippets/:id/dislike
func ToggleDislikeSnippet(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	snippetID := c.Param("id")

	var disliked bool
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var dislike models.SnippetDislike
		result := tx.Where("user_id = ? AND snippet_id = ?", userID, snippetID).First(&dislike)

		if result.Error == nil {
			// Dislike exists -> Remove Dislike
			if err := tx.Delete(&dislike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).UpdateColumn("dislikes_count", gorm.Expr("GREATEST(dislikes_count - 1, 0)")).Error; err != nil {
				return err
			}
			disliked = false
		} else {
			// Dislike doesn't exist -> Dislike

			// 1. Remove Like if exists (Mutually Exclusive)
			var like models.SnippetLike
			if err := tx.Where("user_id = ? AND snippet_id = ?", userID, snippetID).First(&like).Error; err == nil {
				if err := tx.Delete(&like).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error; err != nil {
					return err
				}
			}

			// 2. Create Dislike
			newDislike := models.SnippetDislike{
				UserID:    userID.(string),
				SnippetID: snippetID,
			}
			if err := tx.Create(&newDislike).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).UpdateColumn("dislikes_count", gorm.Expr("dislikes_count + ?", 1)).Error; err != nil {
				return err
			}

			disliked = true
		}
		return nil
	})

	if err != nil {
		fmt.Printf("[Social] ToggleDislikeSnippet FAILED for snippet %s, user %s: %v\n", snippetID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle dislike", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"disliked": disliked})
}

// CheckSnippetLike handles GET /snippets/:id/like
func CheckSnippetLike(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	snippetID := c.Param("id")

	var likeCount int64
	database.DB.Model(&models.SnippetLike{}).Where("user_id = ? AND snippet_id = ?", userID, snippetID).Count(&likeCount)

	var dislikeCount int64
	database.DB.Model(&models.SnippetDislike{}).Where("user_id = ? AND snippet_id = ?", userID, snippetID).Count(&dislikeCount)

	c.JSON(http.StatusOK, gin.H{"liked": likeCount > 0, "disliked": dislikeCount > 0})
}

// --- Comments ---

// AddComment handles POST /snippets/:id/comments
func AddComment(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	snippetID := c.Param("id")

	var input struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := models.Comment{
		UserID:    userID.(string),
		SnippetID: snippetID,
		Content:   input.Content,
	}

	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post comment"})
		return
	}

	// Preload User for immediate display
	database.DB.Preload("User").First(&comment, "id = ?", comment.ID)

	// Create Notification Async
	go func() {
		var snippet models.Snippet
		if err := database.DB.Select("\"authorId\"", "title").First(&snippet, "id = ?", snippetID).Error; err == nil {
			if snippet.AuthorID != userID.(string) {
				notification := models.Notification{
					UserID:    snippet.AuthorID,
					ActorID:   userID.(string),
					Type:      models.NotificationTypeComment,
					SnippetID: &snippetID,
					CommentID: &comment.ID,
					Message:   "commented on your snippet: " + snippet.Title,
				}
				CreateNotification(database.DB, notification)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"comment": comment})
}

// GetSnippetComments handles GET /snippets/:id/comments
func GetSnippetComments(c *gin.Context) {
	snippetID := c.Param("id")

	var comments []models.Comment
	if err := database.DB.Preload("User").Where("snippet_id = ?", snippetID).Order("created_at asc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

// DeleteComment handles DELETE /comments/:id
func DeleteComment(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	commentID := c.Param("id")

	// Ensure user owns comment
	var comment models.Comment
	if err := database.DB.First(&comment, "id = ?", commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.UserID != userID.(string) {
		// Optional: Allow Admin/Moderator to delete?
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
		return
	}

	if err := database.DB.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}

// BlockUser handles POST /users/:username/block
func BlockUser(c *gin.Context) {
	blockerID := c.MustGet("userId").(string)
	targetID := c.Param("username")

	if blockerID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot block yourself"})
		return
	}

	block := models.UserBlock{
		BlockerID: blockerID,
		BlockedID: targetID,
	}

	if err := database.DB.Create(&block).Error; err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusOK, gin.H{"message": "User already blocked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	// Optional: Auto-unlink on block
	database.DB.Delete(&models.UserLink{}, "(linker_id = ? AND linked_id = ?) OR (linker_id = ? AND linked_id = ?)", blockerID, targetID, targetID, blockerID)

	c.JSON(http.StatusOK, gin.H{"message": "User blocked"})
}

// UnblockUser handles DELETE /users/:username/block
func UnblockUser(c *gin.Context) {
	blockerID := c.MustGet("userId").(string)
	targetID := c.Param("username")

	if err := database.DB.Delete(&models.UserBlock{}, "blocker_id = ? AND blocked_id = ?", blockerID, targetID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked"})
}

// GetBlockedUsers GET /users/me/blocks
func GetBlockedUsers(c *gin.Context) {
	userID := c.MustGet("userId").(string)

	var blocks []models.UserBlock
	if err := database.DB.Where("blocker_id = ?", userID).Find(&blocks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blocked users"})
		return
	}

	blockedIDs := make([]string, len(blocks))
	for i, b := range blocks {
		blockedIDs[i] = b.BlockedID
	}

	var blockedUsers []models.User
	if len(blockedIDs) > 0 {
		database.DB.Where("id IN ?", blockedIDs).Find(&blockedUsers)
	} else {
		blockedUsers = []models.User{}
	}

	c.JSON(http.StatusOK, gin.H{"blocked": blockedUsers})
}

// ReportTarget handles POST /users/report
func ReportTarget(c *gin.Context) {
	reporterID := c.MustGet("userId").(string)

	var input struct {
		TargetID   string `json:"targetId" binding:"required"`
		TargetType string `json:"targetType" binding:"required"` // USER, SNIPPET
		Reason     string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report := models.Report{
		ReporterID: reporterID,
		TargetID:   input.TargetID,
		TargetType: input.TargetType,
		Reason:     input.Reason,
		Status:     "PENDING",
	}

	if err := database.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}
