package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"gorm.io/gorm"
)

type CreatePlaylistInput struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	Difficulty  string `json:"difficulty"`
}

type UpdatePlaylistInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	Difficulty  string `json:"difficulty"`
	IsPublished bool   `json:"isPublished"`
}

type AddSnippetInput struct {
	SnippetID string `json:"snippetId" binding:"required"`
	Order     int    `json:"order"`
}

// CreatePlaylist handles POST /playlists
func CreatePlaylist(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input CreatePlaylistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playlist := models.Playlist{
		ID:          utils.GenerateID(),
		Title:       input.Title,
		Description: input.Description,
		Thumbnail:   input.Thumbnail,
		Difficulty:  input.Difficulty,
		AuthorID:    userID.(string),
	}

	if result := database.DB.Create(&playlist); result.Error != nil {
		if strings.Contains(result.Error.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "A playlist with this title already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"playlist": playlist})
}

// GetPlaylist handles GET /playlists/:id
func GetPlaylist(c *gin.Context) {
	id := c.Param("id")
	var playlist models.Playlist

	if err := database.DB.Preload("Author").Preload("Items.Snippet").Preload("Items.Snippet.Author").First(&playlist, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	// Increment views
	database.DB.Model(&playlist).UpdateColumn("views_count", gorm.Expr("views_count + ?", 1))

	c.JSON(http.StatusOK, gin.H{"playlist": playlist})
}

// ListPlaylists handles GET /playlists
func ListPlaylists(c *gin.Context) {
	var playlists []models.Playlist
	query := database.DB.Model(&models.Playlist{}).Preload("Author")

	// Filter by search
	search := c.Query("search")
	if search != "" {
		searchLike := "%" + search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchLike, searchLike)
	}

	// Filter by author
	authorID := c.Query("authorId")
	if authorID != "" {
		query = query.Where("\"authorId\" = ?", authorID)
	}

	// Only published or if it's the author
	// Visibility Logic:
	// 1. If filtering by specific author:
	//    - If author is me: Show all.
	//    - If author is other: Show only published.
	// 2. If global feed (no author filter):
	//    - Show published OR my own drafts.
	currentUserID, hasAuth := c.Get("userId")

	if authorID != "" {
		// Specific author requested
		if !hasAuth || authorID != currentUserID {
			query = query.Where("is_published = ?", true)
		}
	} else {
		// Global feed
		if hasAuth {
			query = query.Where("is_published = ? OR \"authorId\" = ?", true, currentUserID)
		} else {
			query = query.Where("is_published = ?", true)
		}
	}

	// Model has column:createdAt tag, so we must quote it to match case sensitivity
	if err := query.Order("\"createdAt\" DESC").Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

// AddSnippetToPlaylist handles POST /playlists/:id/snippets
func AddSnippetToPlaylist(c *gin.Context) {
	playlistID := c.Param("id")
	userID, _ := c.Get("userId")

	var playlist models.Playlist
	if err := database.DB.First(&playlist, "id = ?", playlistID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if playlist.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to edit this playlist"})
		return
	}

	var input AddSnippetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if already exists
	var existing models.PlaylistSnippet
	if err := database.DB.Where("playlist_id = ? AND snippet_id = ?", playlistID, input.SnippetID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Snippet already in playlist"})
		return
	}

	// Get count for default order
	var count int64
	database.DB.Model(&models.PlaylistSnippet{}).Where("playlist_id = ?", playlistID).Count(&count)

	item := models.PlaylistSnippet{
		ID:         utils.GenerateID(),
		PlaylistID: playlistID,
		SnippetID:  input.SnippetID,
		Order:      int(count),
	}

	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add snippet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Snippet added successfully", "item": item})
}

// ReorderPlaylist handles POST /playlists/:id/reorder
func ReorderPlaylist(c *gin.Context) {
	playlistID := c.Param("id")
	userID, _ := c.Get("userId")

	var playlist models.Playlist
	if err := database.DB.First(&playlist, "id = ?", playlistID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if playlist.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to edit this playlist"})
		return
	}

	var req struct {
		Orders []struct {
			ID    string `json:"id"`
			Order int    `json:"order"`
		} `json:"orders"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, o := range req.Orders {
			if err := tx.Model(&models.PlaylistSnippet{}).Where("id = ? AND playlist_id = ?", o.ID, playlistID).Update("order", o.Order).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist reordered"})
}

// DeletePlaylist handles DELETE /playlists/:id
func DeletePlaylist(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userId")

	var playlist models.Playlist
	if err := database.DB.First(&playlist, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if playlist.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	if err := database.DB.Delete(&playlist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist deleted"})
}

// ClaimEndorsement handles POST /playlists/:id/claim
func ClaimEndorsement(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var playlist models.Playlist
	if err := database.DB.First(&playlist, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if !playlist.IsVerified || playlist.AwardsEndorsement == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This track does not award endorsements"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if already endorsed
	for _, e := range user.Endorsements {
		if e == playlist.AwardsEndorsement {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Endorsement already claimed"})
			return
		}
	}

	// Add endorsement and grant XP
	user.Endorsements = append(user.Endorsements, playlist.AwardsEndorsement)
	user.XP += 250 // Completion bonus

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim endorsement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Certification claimed successfully!",
		"endorsement":  playlist.AwardsEndorsement,
		"xp":           user.XP,
		"endorsements": user.Endorsements,
	})
}
