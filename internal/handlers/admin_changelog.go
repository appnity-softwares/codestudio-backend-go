package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// AdminListChangelogs returns all entries (drafts & published)
func AdminListChangelogs(c *gin.Context) {
	var entries []models.ChangelogEntry
	// Order by "order" ascending (or could be descending, but typical DND uses asc order values)
	if err := database.DB.Order("\"order\" ASC, created_at DESC").Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch changelogs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// AdminCreateChangelog creates a new draft release
func AdminCreateChangelog(c *gin.Context) {
	adminID := getAdminID(c)
	var req struct {
		Version     string     `json:"version" binding:"required"`
		Title       string     `json:"title" binding:"required"`
		Description string     `json:"description" binding:"required"`
		ReleaseType string     `json:"releaseType"`
		ReleasedAt  *time.Time `json:"releasedAt"`
		Order       int        `json:"order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry := models.ChangelogEntry{
		ID:          uuid.New().String(),
		Version:     req.Version,
		Title:       req.Title,
		Description: req.Description,
		ReleaseType: req.ReleaseType,
		IsPublished: false,
		ReleasedAt:  req.ReleasedAt,
		Order:       req.Order,
		CreatedAt:   time.Now(),
		CreatedBy:   adminID,
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create changelog"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, entry.ID, "changelog", "Created Draft: "+entry.Version) // Reusing ActionAdjustTrust as generic edit for now or new type? Using generic log.

	c.JSON(http.StatusOK, gin.H{"entry": entry, "message": "Draft created"})
}

// AdminUpdateChangelog updates an existing entry
func AdminUpdateChangelog(c *gin.Context) {
	id := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Version     string     `json:"version"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		ReleaseType string     `json:"releaseType"`
		IsPublished bool       `json:"isPublished"`
		ReleasedAt  *time.Time `json:"releasedAt"`
		Order       int        `json:"order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var entry models.ChangelogEntry
		if err := tx.First(&entry, "id = ?", id).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"version":      req.Version,
			"title":        req.Title,
			"description":  req.Description,
			"release_type": req.ReleaseType,
			"is_published": req.IsPublished,
			"order":        req.Order,
		}

		if req.ReleasedAt != nil {
			updates["released_at"] = req.ReleasedAt
		}

		// Handle publish state change
		if req.IsPublished && !entry.IsPublished {
			now := time.Now()
			updates["released_at"] = &now
		} else if !req.IsPublished {
			updates["released_at"] = nil
		}

		if err := tx.Model(&entry).Updates(updates).Error; err != nil {
			return err
		}

		action := "Updated changelog"
		if req.IsPublished != entry.IsPublished {
			if req.IsPublished {
				action = "Published changelog"
			} else {
				action = "Unpublished changelog"
			}
		}

		return logAdminAction(tx, adminID, models.ActionAdjustTrust, id, "changelog", action)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Changelog updated"})
}

// AdminDeleteChangelog deletes an entry
func AdminDeleteChangelog(c *gin.Context) {
	id := c.Param("id")
	adminID := getAdminID(c)

	if err := database.DB.Delete(&models.ChangelogEntry{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionAdjustTrust, id, "changelog", "Deleted changelog")

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// AdminReorderChangelogs handles bulk order updates
func AdminReorderChangelogs(c *gin.Context) {
	adminID := getAdminID(c)
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
			if err := tx.Model(&models.ChangelogEntry{}).Where("id = ?", o.ID).Update("order", o.Order).Error; err != nil {
				return err
			}
		}
		return logAdminAction(tx, adminID, models.ActionAdjustTrust, "bulk", "changelog", "Reordered changelogs")
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orders updated"})
}
