package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// AdminListRoadmaps returns paginated roadmaps with filters
func AdminListRoadmaps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	filter := c.Query("filter") // all, verified, unverified

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Playlist{}).Preload("Author").Preload("Items")

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	switch filter {
	case "verified":
		query = query.Where("is_verified = ?", true)
	case "unverified":
		query = query.Where("is_verified = ?", false)
	}

	var total int64
	query.Count(&total)

	var playlists []models.Playlist
	if err := query.Order("\"createdAt\" desc").Offset(offset).Limit(limit).Find(&playlists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roadmaps"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roadmaps": playlists,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// AdminVerifyRoadmap toggles verification status and sets bonus XP
func AdminVerifyRoadmap(c *gin.Context) {
	playlistID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		IsVerified        bool   `json:"isVerified"`
		AwardsEndorsement string `json:"awardsEndorsement"`
		CompletionBonusXP int    `json:"completionBonusXP"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var playlist models.Playlist
		if err := tx.First(&playlist, "id = ?", playlistID).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"is_verified":         req.IsVerified,
			"awards_endorsement":  req.AwardsEndorsement,
			"completion_bonus_xp": req.CompletionBonusXP,
		}

		if err := tx.Model(&playlist).Updates(updates).Error; err != nil {
			return err
		}

		action := "Unverified Roadmap"
		if req.IsVerified {
			action = "Verified Roadmap (Bonus: " + strconv.Itoa(req.CompletionBonusXP) + " XP)"
		}

		return logAdminAction(tx, adminID, models.ActionUpdateContest, playlistID, "playlist", action)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Roadmap verification updated"})
}

// AdminDeleteRoadmap deletes a roadmap
func AdminDeleteRoadmap(c *gin.Context) {
	playlistID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("playlist_id = ?", playlistID).Delete(&models.PlaylistSnippet{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Playlist{}, "id = ?", playlistID).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionDeleteContest, playlistID, "playlist", "Deleted by Admin")
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Roadmap deleted"})
}
