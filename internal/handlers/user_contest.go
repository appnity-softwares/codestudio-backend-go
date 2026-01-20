package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// GetMyContestHistory handles GET /users/me/contests
func GetMyContestHistory(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var registrations []models.Registration
	if err := database.DB.Preload("Event").
		Joins("JOIN events ON events.id = registrations.event_id").
		Where("registrations.user_id = ? AND events.\"isExternal\" = ?", userID, false).
		Order("events.start_time desc").
		Find(&registrations).Error; err != nil {
		fmt.Printf("Error fetching contest history for user %s: %v\n", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch contest history: %v", err)})
		return
	}

	var history []gin.H
	for _, reg := range registrations {
		// Only include finished contests or joined ones
		history = append(history, gin.H{
			"id":            reg.Event.ID,
			"title":         reg.Event.Title,
			"rank":          reg.Rank,
			"score":         reg.Score,
			"status":        reg.Event.Status, // LIVE, ENDED, etc.
			"startTime":     reg.Event.StartTime,
			"endTime":       reg.Event.EndTime,
			"rulesAccepted": reg.RulesAccepted,
		})
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}
