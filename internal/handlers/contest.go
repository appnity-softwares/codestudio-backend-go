package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/services"
)

// GetContestLeaderboard returns the calculated leaderboard for an event
func GetContestLeaderboard(c *gin.Context) {
	eventID := c.Param("eventId")

	// Optional: Check if user is registered?
	// Leaderboards are often public, or at least public to platform users.

	// Check if Admin (to bypass freeze)
	asAdmin := false
	if role, exists := c.Get("role"); exists {
		if r, ok := role.(string); ok && r == "ADMIN" {
			asAdmin = true
		}
	}

	leaderboard, err := services.GetLeaderboard(eventID, asAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leaderboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"eventId":     eventID,
		"leaderboard": leaderboard,
	})
}
