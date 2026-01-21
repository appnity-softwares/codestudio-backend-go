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
			Icon:        "code",
			Category:    models.BadgeCategorySkill,
			Condition:   "1_snippet",
			Threshold:   1,
		},
		{
			Name:        "Snippet Enthusiast",
			Description: "Published 5 code snippets to the feed.",
			Icon:        "terminal",
			Category:    models.BadgeCategorySkill,
			Condition:   "5_snippets",
			Threshold:   5,
		},
		{
			Name:        "Snippet Master",
			Description: "Published 25 snippets. A true code architect.",
			Icon:        "crown",
			Category:    models.BadgeCategorySkill,
			Condition:   "25_snippets",
			Threshold:   25,
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
			Name:        "Apprentice Solver",
			Description: "Solved 5 practice problems.",
			Icon:        "zap",
			Category:    models.BadgeCategorySkill,
			Condition:   "5_practice_solved",
			Threshold:   5,
		},
		{
			Name:        "Algorithm Architect",
			Description: "Solved 25 practice problems.",
			Icon:        "shield-check",
			Category:    models.BadgeCategorySkill,
			Condition:   "25_practice_solved",
			Threshold:   25,
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
			Name:        "Arena Veteran",
			Description: "Competed in 5 official Arena events.",
			Icon:        "sword",
			Category:    models.BadgeCategorySkill,
			Condition:   "5_contests",
			Threshold:   5,
		},
		{
			Name:        "Early Adopter",
			Description: "Joined during the initial launch phase.",
			Icon:        "star",
			Category:    models.BadgeCategoryTrust,
			Condition:   "early_adopter",
			Threshold:   1,
		},
		{
			Name:        "Feedback Contributor",
			Description: "Helping improve the platform.",
			Icon:        "message-square",
			Category:    models.BadgeCategoryTrust,
			Condition:   "feedback_given",
			Threshold:   1,
		},
		{
			Name:        "Community Pillar",
			Description: "Provided 5 pieces of constructive feedback.",
			Icon:        "heart",
			Category:    models.BadgeCategoryTrust,
			Condition:   "5_feedback",
			Threshold:   5,
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
