package handlers

import (
	"net/http"
	"time"

	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// AdminAddAvatarSeed adds a new avatar seed
func AdminAddAvatarSeed(c *gin.Context) {
	adminID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Seed  string `json:"seed" binding:"required"`
		Style string `json:"style"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Style == "" {
		input.Style = "avataaars"
	}

	avatar := models.AvatarSeed{
		Seed:      input.Seed,
		Style:     input.Style,
		AddedBy:   adminID.(string),
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&avatar).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add avatar seed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Avatar seed added", "avatar": avatar})
}

// AdminDeleteAvatarSeed removes an avatar seed
func AdminDeleteAvatarSeed(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.AvatarSeed{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete avatar seed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Avatar seed deleted"})
}

// GetAvatarSeeds returns all available avatar seeds
func GetAvatarSeeds(c *gin.Context) {
	var seeds []models.AvatarSeed
	if err := database.DB.Find(&seeds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch avatars"})
		return
	}

	// Default seeds if none in DB
	if len(seeds) == 0 {
		defaults := []string{"Felix", "Aneka", "Mason", "Jude", "Clara", "Lilly", "Max", "Toby"}
		for i, s := range defaults {
			seeds = append(seeds, models.AvatarSeed{
				ID:    uint(i + 1),
				Seed:  s,
				Style: "avataaars",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"avatars": seeds})
}
