package seeds

import (
	"log"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedBadges() {
	log.Println("🎖️ Seeding System Badges...")

	badges := []models.Badge{
		{
			Name:        "First Snippet",
			Description: "Created your first code snippet.",
			Icon:        "code", // Lucide icon name
			Category:    models.BadgeCategorySkill,
			Condition:   "1_snippet",
			Threshold:   1,
		},
		{
			Name:        "Problem Solver",
			Description: "Solved your first practice problem.",
			Icon:        "check-circle",
			Category:    models.BadgeCategorySkill,
			Condition:   "1_practice_solved",
			Threshold:   1,
		},
		{
			Name:        "Contest Participant",
			Description: "Participated in an official contest.",
			Icon:        "trophy",
			Category:    models.BadgeCategorySkill,
			Condition:   "1_contest",
			Threshold:   1,
		},
		{
			Name:        "Early Adopter",
			Description: "Joined during the initial launch phase.",
			Icon:        "star",
			Category:    models.BadgeCategoryTrust,
			Condition:   "early_adopter",
			Threshold:   0,
		},
		{
			Name:        "Feedback Contributor",
			Description: "Helping improve the platform.",
			Icon:        "message-square",
			Category:    models.BadgeCategoryTrust,
			Condition:   "feedback_given",
			Threshold:   1,
		},
	}

	for _, b := range badges {
		var existing models.Badge
		if err := database.DB.Where("name = ?", b.Name).First(&existing).Error; err == nil {
			log.Printf("   ℹ️ Badge already exists: %s", b.Name)
			continue
		}

		b.ID = uuid.New().String()
		if err := database.DB.Create(&b).Error; err != nil {
			log.Printf("   ❌ Failed to create badge %s: %v", b.Name, err)
		} else {
			log.Printf("   🎖️ Badge Defined: %s", b.Name)
		}
	}
}
