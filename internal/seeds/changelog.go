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
	entries := []models.ChangelogEntry{
		{
			ID:        uuid.New().String(),
			Version:   "1.0.0-MVP",
			Title:     "Initial Launch",
			Changes:   []string{"Snippet sharing & execution", "Arena contests", "Socket.io presence", "Basic profile metrics"},
			CreatedAt: time.Now().Add(-72 * time.Hour),
		},
		{
			ID:        uuid.New().String(),
			Version:   "1.1.0",
			Title:     "Rich Snippets",
			Changes:   []string{"Snippet types & difficulty", "Enhanced code previews", "Output snapshots"},
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			ID:        uuid.New().String(),
			Version:   "1.2.0",
			Title:     "Smart Feed & Progress",
			Changes:   []string{"Smart Feed (Trending/New/Editor)", "Fork & Copy snippets", "User progress stats", "Trust score & ranking", "Admin snippet pinning"},
			CreatedAt: time.Now(),
		},
	}

	for _, e := range entries {
		database.DB.Create(&e)
	}
}
