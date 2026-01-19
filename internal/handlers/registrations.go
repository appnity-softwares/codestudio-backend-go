package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// GetMyRegistrations handles GET /registrations/my
func GetMyRegistrations(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var registrations []models.Registration
	if result := database.DB.Preload("Event").Where("user_id = ?", userID).Find(&registrations); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch registrations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registrations": registrations})
}

// ListRegistrations handles GET /registrations (Admin)
func ListRegistrations(c *gin.Context) {
	// TODO: Add Admin Middleware Check
	// For now, assuming route is protected by simple auth, but needing role check ideally.

	status := c.Query("status")
	eventId := c.Query("eventId")

	query := database.DB.Preload("User").Preload("Event")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if eventId != "" {
		query = query.Where("event_id = ?", eventId)
	}

	var registrations []models.Registration
	if result := query.Order("created_at desc").Find(&registrations); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch registrations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registrations": registrations})
}

// UpdateRegistrationStatus handles PATCH /registrations/:id/status (Admin)
func UpdateRegistrationStatus(c *gin.Context) {
	id := c.Param("id")

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var registration models.Registration
	if result := database.DB.First(&registration, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
		return
	}

	registration.Status = models.RegistrationStatus(input.Status)
	if err := database.DB.Save(&registration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"registration": registration})
}
