package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// Helper to get a system setting value
func getSystemSetting(key string) string {
	var setting models.SystemSettings
	if err := database.DB.Where("key = ?", key).Limit(1).Find(&setting).Error; err != nil {
		return "" // Return empty on actual DB error
	}
	return setting.Value
}

// MaintenanceMode blocks all non-admin users when maintenance mode is enabled
func MaintenanceMode() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if maintenance mode is enabled
		if getSystemSetting(models.SettingMaintenanceMode) != "true" {
			c.Next()
			return
		}

		// Always allow profile check so frontend can determine if user is admin
		if c.Request.URL.Path == "/api/users/profile" {
			c.Next()
			return
		}

		// Allow admin users to pass through
		userID, exists := c.Get("userId")
		if exists {
			var user models.User
			if err := database.DB.First(&user, "id = ?", userID.(string)).Error; err == nil {
				if user.Role == models.RoleAdmin {
					c.Next()
					return
				}
			}
		}

		// Block non-admin users
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Maintenance in progress",
			"message": "The platform is currently under maintenance. Please try again later.",
			"eta":     getSystemSetting(models.SettingMaintenanceETA),
		})
		c.Abort()
	}
}

// RequireSubmissionsEnabled blocks submission endpoints when disabled
func RequireSubmissionsEnabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if getSystemSetting(models.SettingSubmissionsEnabled) == "false" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Submissions are currently disabled",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireSnippetsEnabled blocks snippet creation when disabled
func RequireSnippetsEnabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if getSystemSetting(models.SettingSnippetsEnabled) == "false" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Snippet creation is currently disabled",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireContestsEnabled blocks contest registration when disabled
func RequireContestsEnabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if getSystemSetting(models.SettingContestsEnabled) == "false" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Contests are currently disabled",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRegistrationOpen blocks user registration when disabled
func RequireRegistrationOpen() gin.HandlerFunc {
	return func(c *gin.Context) {
		if getSystemSetting(models.SettingRegistrationOpen) == "false" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "User registration is currently closed",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// FeatureGate blocks access to a feature if its toggle is disabled
func FeatureGate(key string, featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !database.IsFeatureEnabled(key) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Feature Disabled",
				"message": featureName + " is currently disabled by administrators.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
