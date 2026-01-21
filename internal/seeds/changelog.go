package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedChangelog() {
	log.Println("📢 Seeding Changelog...")

	now := time.Now()
	entry := models.ChangelogEntry{
		Version: "v1.0",
		Title:   "CodeStudio Initial Launch",
		Description: `We are excited to launch CodeStudio! 🚀

**New Features:**
- **Code Snippets:** Share and discover useful code fragments.
- **Practice Arena:** Sharpen your skills with algorithmic problems.
- **Contests:** Compete in official events.
- **Community:** Public profiles and feedback wall.

Enjoy coding!`,
		ReleaseType: "FEATURE", // Using FEATURE as a general type for launch
		IsPublished: true,
		ReleasedAt:  &now,
		CreatedAt:   now,
	}

	var existing models.ChangelogEntry
	if err := database.DB.Where("version = ?", entry.Version).First(&existing).Error; err == nil {
		log.Printf("   ℹ️ Changelog %s already exists", entry.Version)
		return
	}

	entry.ID = uuid.New().String()
	if err := database.DB.Create(&entry).Error; err != nil {
		log.Printf("   ❌ Failed to seed changelog: %v", err)
	} else {
		log.Printf("   📢 Changelog Published: %s", entry.Version)
	}
}
