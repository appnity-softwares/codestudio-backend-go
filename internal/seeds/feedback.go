package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedFeedback(systemUser models.User) {
	log.Println("💬 Seeding Feedback Wall...")

	feedbacks := []models.FeedbackMessage{
		{
			Content:  "Welcome to CodeStudio 👋\nThis is an early version. Your feedback directly shapes what we build next.",
			Category: models.CategoryOther,
			IsPinned: true,
			IsLocked: true, // Announcement
			Status:   models.StatusOpen,
		},
		{
			Content:  "Use this wall to:\n- Report issues\n- Suggest features\n- Share confusion\nWe read everything.",
			Category: models.CategoryFeature,
			IsPinned: true,
			IsLocked: false,
			Status:   models.StatusOpen,
		},
	}

	for _, f := range feedbacks {
		var existing models.FeedbackMessage
		// Check for duplicate pinned messages by content prefix (rudimentary check) or exact content
		if err := database.DB.Where("content = ? AND is_pinned = ?", f.Content, true).First(&existing).Error; err == nil {
			log.Printf("   ℹ️ Feedback post already exists")
			continue
		}

		f.ID = uuid.New().String()
		f.UserID = systemUser.ID
		f.CreatedAt = time.Now()
		f.Upvotes = 10 // Start with some love

		if err := database.DB.Create(&f).Error; err != nil {
			log.Printf("   ❌ Failed to create feedback: %v", err)
		} else {
			log.Printf("   📌 Feedback Pinned: %s...", f.Content[:20])
		}
	}
}
