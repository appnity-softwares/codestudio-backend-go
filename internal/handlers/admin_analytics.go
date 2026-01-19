package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// ============================================
// ANALYTICS
// ============================================

// AdminGetTopSnippets returns snippets ordered by view count
func AdminGetTopSnippets(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	var snippets []models.Snippet
	if err := database.DB.Preload("Author").Order("views_count desc").Limit(limit).Find(&snippets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top snippets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippets": snippets})
}

// AdminGetSuspiciousActivity returns snippets with high fork/copy activity
func AdminGetSuspiciousActivity(c *gin.Context) {
	// Snippets with high copy counts (potential IP theft or popular utility)
	var highCopySnippets []models.Snippet
	database.DB.Preload("Author").Order("copy_count desc").Limit(10).Find(&highCopySnippets)

	// Snippets with high fork counts (viral or spam)
	var highForkSnippets []models.Snippet
	database.DB.Preload("Author").Order("fork_count desc").Limit(10).Find(&highForkSnippets)

	c.JSON(http.StatusOK, gin.H{
		"highCopySnippets": highCopySnippets,
		"highForkSnippets": highForkSnippets,
	})
}
