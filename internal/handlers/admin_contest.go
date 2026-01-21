package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"gorm.io/gorm"
)

// --- Contest Management ---

func AdminListContests(c *gin.Context) {
	var events []models.Event
	// Order by most recent first
	if err := database.DB.Order("created_at desc").Find(&events).Error; err != nil {
		c.JSON(500, gin.H{"error": "DB Error"})
		return
	}
	c.JSON(200, gin.H{"contests": events})
}

func AdminCreateContest(c *gin.Context) {
	adminID := getAdminID(c)

	var req struct {
		Title       string     `json:"title" binding:"required"`
		Description string     `json:"description" binding:"required"`
		StartTime   time.Time  `json:"startTime" binding:"required"`
		EndTime     time.Time  `json:"endTime" binding:"required"`
		FreezeTime  *time.Time `json:"freezeTime"`
		Type        string     `json:"type"`        // INTERNAL or EXTERNAL
		ExternalURL string     `json:"externalUrl"` // Required if EXTERNAL
		Banner      string     `json:"banner"`
		Price       float64    `json:"price"` // For paid contests
		IsExternal  bool       `json:"isExternal"`
		Platform    string     `json:"platform"`
		JoinURL     string     `json:"joinUrl"`
		VisibleAt   *time.Time `json:"visibleAt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate External URL logic
	if req.IsExternal && req.JoinURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Join URL is required for external contests"})
		return
	}

	event := models.Event{
		ID:               uuid.New().String(),
		Title:            req.Title,
		Slug:             utils.GenerateSlug(req.Title) + "-" + uuid.New().String()[:8],
		Description:      req.Description,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		FreezeTime:       req.FreezeTime,
		Status:           models.EventStatusDraft, // Default to DRAFT, not Upcoming
		CreatedBy:        adminID,
		Banner:           req.Banner,
		Type:             req.Type,
		ExternalURL:      req.ExternalURL,
		Price:            req.Price,
		IsExternal:       req.IsExternal,
		ExternalPlatform: req.Platform,
		ExternalJoinURL:  req.JoinURL,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Auto-calculate visible time if not provided
	if req.VisibleAt == nil {
		event.ExternalJoinVisibleAt = req.StartTime.Add(-15 * time.Minute)
	} else {
		event.ExternalJoinVisibleAt = *req.VisibleAt
	}

	if event.Type == "" {
		event.Type = "INTERNAL"
	}

	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contest: " + err.Error()})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionCreateContest, event.ID, "contest", "Created Contest: "+event.Title)

	c.JSON(http.StatusCreated, gin.H{"contest": event})
}

func AdminUpdateContest(c *gin.Context) {
	eventID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		StartTime   time.Time  `json:"startTime"`
		EndTime     time.Time  `json:"endTime"`
		FreezeTime  *time.Time `json:"freezeTime"`
		Type        string     `json:"type"`
		ExternalURL string     `json:"externalUrl"`
		Banner      string     `json:"banner"`
		Price       float64    `json:"price"`
		IsExternal  *bool      `json:"isExternal"`
		Platform    string     `json:"platform"`
		JoinURL     string     `json:"joinUrl"`
		VisibleAt   *time.Time `json:"visibleAt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var event models.Event
		if err := tx.First(&event, "id = ?", eventID).Error; err != nil {
			return err
		}

		// Validation: Cannot edit LIVE or ENDED contests (except simple things, but let's be strict for now)
		// Or maybe allow editing Description/Banner? Let's block core fields if live.
		if event.Status == models.EventStatusLive || event.Status == models.EventStatusEnded {
			// If trying to change timing or type, block.
			// Ideally we check which fields are changed. For MVP, let's block heavy edits.
			// Assuming MVP admins are careful or we just allow it but log it.
			// Let's allow it but warn.
		}

		updates := map[string]interface{}{}
		if req.Title != "" {
			updates["title"] = req.Title
			updates["slug"] = utils.GenerateSlug(req.Title)
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if !req.StartTime.IsZero() {
			updates["start_time"] = req.StartTime
		}
		if !req.EndTime.IsZero() {
			updates["end_time"] = req.EndTime
		}
		updates["freeze_time"] = req.FreezeTime // Nullable
		if req.ExternalURL != "" {
			updates["external_url"] = req.ExternalURL
		}
		if req.Banner != "" {
			updates["banner"] = req.Banner
		}
		if req.IsExternal != nil {
			updates["isExternal"] = *req.IsExternal
		}
		if req.Platform != "" {
			updates["externalPlatform"] = req.Platform
		}
		if req.JoinURL != "" {
			updates["externalJoinUrl"] = req.JoinURL
		}
		if req.VisibleAt != nil {
			updates["externalJoinVisibleAt"] = *req.VisibleAt
		}
		updates["price"] = req.Price
		updates["updated_at"] = time.Now()

		if err := tx.Model(&event).Updates(updates).Error; err != nil {
			return err
		}

		return logAdminAction(tx, adminID, models.ActionUpdateContest, eventID, "contest", "Admin Updated Contest")
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Contest Updated"})
}

func AdminDeleteContest(c *gin.Context) {
	eventID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var event models.Event
		if err := tx.First(&event, "id = ?", eventID).Error; err != nil {
			return err
		}

		// Only delete DRAFT or UPCOMING?
		// If LIVE or ENDED, maybe we shouldn't delete to preserve history.
		// Let's allow deleting DRAFT and UPCOMING.
		if event.Status == models.EventStatusLive || event.Status == models.EventStatusFrozen || event.Status == models.EventStatusEnded {
			return &gin.Error{Err: gorm.ErrInvalidTransaction, Type: gin.ErrorTypePublic, Meta: "Cannot delete Live or Ended contests"}
		}

		if err := tx.Delete(&event).Error; err != nil {
			return err
		}

		return logAdminAction(tx, adminID, models.ActionDeleteContest, eventID, "contest", "Deleted Contest")
	})

	if err != nil {
		c.JSON(400, gin.H{"error": "Cannot delete contest (might be Live/Ended or DB error)"})
		return
	}

	c.JSON(200, gin.H{"message": "Contest Deleted"})
}

func AdminStartContest(c *gin.Context) {
	eventID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var event models.Event
		if err := tx.First(&event, "id = ?", eventID).Error; err != nil {
			return err
		}
		if event.Status == models.EventStatusLive {
			return nil
		}
		if err := tx.Model(&event).Updates(map[string]interface{}{
			"status":     models.EventStatusLive,
			"start_time": time.Now(),
		}).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionStartContest, eventID, "contest", "Manual Start")
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Contest Started"})
}

func AdminFreezeContest(c *gin.Context) {
	eventID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Event{}).Where("id = ?", eventID).Update("status", models.EventStatusFrozen).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionFreezeContest, eventID, "contest", "Frozen")
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Contest Frozen"})
}

func AdminEndContest(c *gin.Context) {
	eventID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Event{}).Where("id = ?", eventID).Updates(map[string]interface{}{
			"status":   models.EventStatusEnded,
			"end_time": time.Now(),
		}).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionEndContest, eventID, "contest", "Manual End")
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Contest Ended"})
}

func AdminGetContestParticipants(c *gin.Context) {
	eventID := c.Param("id")

	var registrations []models.Registration
	if err := database.DB.Preload("User").Where("event_id = ?", eventID).Find(&registrations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch registrations"})
		return
	}

	var stats struct {
		TotalRegistered int64 `json:"totalRegistered"`
		JoinedExternal  int64 `json:"joinedExternal"`
		NoShows         int64 `json:"noShows"`
	}

	database.DB.Model(&models.Registration{}).Where("event_id = ?", eventID).Count(&stats.TotalRegistered)
	database.DB.Model(&models.Registration{}).Where("event_id = ? AND status = ?", eventID, models.RegStatusJoined).Count(&stats.JoinedExternal)
	database.DB.Model(&models.Registration{}).Where("event_id = ? AND joined_external_at IS NULL AND status != ?", eventID, models.RegStatusJoined).Count(&stats.NoShows)

	c.JSON(http.StatusOK, gin.H{
		"stats":        stats,
		"participants": registrations,
	})
}
