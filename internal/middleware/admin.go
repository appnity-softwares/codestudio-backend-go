package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// StaffOnly middleware allows ADMIN and MODERATOR (if they have permissions)
func StaffOnly(permissionField string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, "id = ?", userID.(string)).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if user.Role == models.RoleAdmin {
			c.Next()
			return
		}

		if user.Role == models.RoleModerator {
			if permissionField == "" {
				c.Next() // Generic staff access
				return
			}

			var perms models.RolePermission
			if err := database.DB.Where("role = ?", models.RoleModerator).First(&perms).Error; err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "Moderator permissions not configured"})
				c.Abort()
				return
			}

			// Use reflection or a switch to check the field
			hasAccess := false
			switch permissionField {
			case "CanManageUsers":
				hasAccess = perms.CanManageUsers
			case "CanManageSnippets":
				hasAccess = perms.CanManageSnippets
			case "CanManageContests":
				hasAccess = perms.CanManageContests
			case "CanManageProblems":
				hasAccess = perms.CanManageProblems
			case "CanViewAuditLogs":
				hasAccess = perms.CanViewAuditLogs
			case "CanManageSystem":
				hasAccess = perms.CanManageSystem
			}

			if hasAccess {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// AdminOnly middleware restricts access to users with ADMIN role only
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, "id = ?", userID.(string)).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if user.Role != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
