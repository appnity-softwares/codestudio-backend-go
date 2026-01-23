package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// ListChangelog returns all changelog entries (public - published only)
func ListChangelog(c *gin.Context) {
	var entries []models.ChangelogEntry
	if err := database.DB.Where("is_published = ?", true).Order("\"order\" ASC, released_at DESC").Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch changelog"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}
