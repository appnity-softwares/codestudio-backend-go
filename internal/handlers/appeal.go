package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// CreateAppeal handles public suspension appeals
func CreateAppeal(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Reason   string `json:"reason" binding:"required,min=20,max=1000"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appeal := models.Appeal{
		Email:     input.Email,
		Username:  input.Username,
		Reason:    input.Reason,
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&appeal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit appeal. Please try again later."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Appeal submitted successfully. Our team will review it and contact you via email.",
	})
}

// AdminGetAppeals lists all appeals for admins
func AdminGetAppeals(c *gin.Context) {
	var appeals []models.Appeal
	if err := database.DB.Order("created_at desc").Find(&appeals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch appeals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"appeals": appeals})
}

// AdminUpdateAppealStatus updates the status of an appeal
func AdminUpdateAppealStatus(c *gin.Context) {
	appealID := c.Param("id")
	var input struct {
		Status string `json:"status" binding:"required"` // PENDING, REVIEWED, RESOLVED
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&models.Appeal{}).Where("id = ?", appealID).Update("status", input.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update appeal status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appeal status updated"})
}
