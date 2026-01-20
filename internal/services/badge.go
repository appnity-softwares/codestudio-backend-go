package services

import (
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// CheckBadges checks if the user has earned any new badges after a submission
// It returns a slice of newly awarded badges
func CheckBadges(userID string) ([]models.Badge, error) {
	var newBadges []models.Badge

	// Get user's current badges
	var existingBadgeIDs []string
	database.DB.Model(&models.UserBadge{}).Where("user_id = ?", userID).Pluck("badge_id", &existingBadgeIDs)

	existingSet := make(map[string]bool)
	for _, id := range existingBadgeIDs {
		existingSet[id] = true
	}

	// Calculate stats
	var solvedCount int64
	database.DB.Model(&models.PracticeSubmission{}).
		Where("user_id = ? AND status = ?", userID, "ACCEPTED").
		Distinct("problem_id").
		Count(&solvedCount)

	// Define rules
	type BadgeRule struct {
		ID          string
		Name        string
		Description string
		Icon        string
		Condition   func() bool
	}

	rules := []BadgeRule{
		{
			ID:          "first-blood",
			Name:        "First Blood",
			Description: "Solved your first practice problem",
			Icon:        "🩸",
			Condition:   func() bool { return solvedCount >= 1 },
		},
		{
			ID:          "solver-5",
			Name:        "Apprentice",
			Description: "Solved 5 practice problems",
			Icon:        "🥉",
			Condition:   func() bool { return solvedCount >= 5 },
		},
		{
			ID:          "solver-10",
			Name:        "Problem Solver",
			Description: "Solved 10 practice problems",
			Icon:        "🥈",
			Condition:   func() bool { return solvedCount >= 10 },
		},
		{
			ID:          "solver-25",
			Name:        "Algorithmist",
			Description: "Solved 25 practice problems",
			Icon:        "🥇",
			Condition:   func() bool { return solvedCount >= 25 },
		},
	}

	// Check rules
	for _, rule := range rules {
		if !existingSet[rule.ID] && rule.Condition() {
			// Award badge
			badge := models.Badge{
				ID:          rule.ID,
				Name:        rule.Name,
				Description: rule.Description,
				Icon:        rule.Icon,
			}

			// Upsert Badge Definition (ensure it exists)
			database.DB.FirstOrCreate(&badge, models.Badge{ID: rule.ID})

			// Create UserBadge
			userBadge := models.UserBadge{
				UserID:  userID,
				BadgeID: rule.ID,
			}
			if err := database.DB.Create(&userBadge).Error; err == nil {
				newBadges = append(newBadges, badge)
			}
		}
	}

	return newBadges, nil
}
