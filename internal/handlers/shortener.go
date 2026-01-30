package handlers

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortCode(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// CreateShortLink handles POST /api/shorten
func CreateShortLink(c *gin.Context) {
	var input struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Simple check to avoid creating new links for same URL repeatedly (optional)
	var existing models.ShortLink
	if err := database.DB.Where("original_url = ?", input.URL).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"originalUrl": existing.OriginalURL,
			"code":        existing.Code,
			"shortUrl":    "/s/" + existing.Code, // Frontend can prepend domain or we return full
		})
		return
	}

	code := generateShortCode(6)
	// Ensure uniqueness (simple retry)
	for i := 0; i < 5; i++ {
		var count int64
		database.DB.Model(&models.ShortLink{}).Where("code = ?", code).Count(&count)
		if count == 0 {
			break
		}
		code = generateShortCode(6)
	}

	link := models.ShortLink{
		Code:        code,
		OriginalURL: input.URL,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(&link).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"originalUrl": link.OriginalURL,
		"code":        link.Code,
		"shortUrl":    "/s/" + link.Code,
	})
}

// RedirectShortLink handles GET /s/:code
func RedirectShortLink(c *gin.Context) {
	code := c.Param("code")

	var link models.ShortLink
	if err := database.DB.Where("code = ?", code).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	// Increment visits async
	go func() {
		database.DB.Model(&link).UpdateColumn("visits", link.Visits+1)
	}()

	c.Redirect(http.StatusFound, link.OriginalURL)
}
