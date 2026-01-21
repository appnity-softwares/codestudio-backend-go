package services

import (
	"time"

	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// CheckBadges checks if the user has earned any new badges after an action
func CheckBadges(userID string) ([]models.Badge, error) {
	var newBadges []models.Badge

	// 1. Get user's current badges
	var existingBadgeIDs []string
	database.DB.Model(&models.UserBadge{}).Where("user_id = ?", userID).Pluck("badge_id", &existingBadgeIDs)

	existingSet := make(map[string]bool)
	for _, id := range existingBadgeIDs {
		existingSet[id] = true
	}

	// 2. Fetch Stats
	var solvedCount int64
	database.DB.Model(&models.Submission{}).Where("user_id = ? AND status = 'ACCEPTED'", userID).Count(&solvedCount)

	var snippetCount int64
	database.DB.Model(&models.Snippet{}).Where("author_id = ? OR \"authorId\" = ?", userID, userID).Count(&snippetCount)

	var feedbackCount int64
	database.DB.Model(&models.FeedbackMessage{}).Where("user_id = ?", userID).Count(&feedbackCount)

	var contestCount int64
	database.DB.Model(&models.Registration{}).Where("user_id = ? AND status != 'BANNED'", userID).Count(&contestCount)

	// Count Early Adopter qualification (First 1000 Users)
	var rank int64
	database.DB.Model(&models.User{}).
		Where("created_at <= (SELECT created_at FROM users WHERE id = ?)", userID).
		Count(&rank)

	earlyAdopterStatus := int64(0)
	if rank <= 1000 {
		earlyAdopterStatus = 1
	}

	// 3. Define mapping of conditions to stats
	stats := map[string]int64{
		"1_snippet":          snippetCount,
		"5_snippets":         snippetCount,
		"25_snippets":        snippetCount,
		"1_practice_solved":  solvedCount,
		"5_practice_solved":  solvedCount,
		"25_practice_solved": solvedCount,
		"feedback_given":     feedbackCount,
		"5_feedback":         feedbackCount,
		"early_adopter":      earlyAdopterStatus,
		"1_contest":          contestCount,
		"5_contests":         contestCount,
	}

	// 4. Fetch all system badge definitions
	var systemBadges []models.Badge
	database.DB.Find(&systemBadges)

	// 5. Evaluate each badge
	for _, badge := range systemBadges {
		// Skip if already owned
		if existingSet[badge.ID] {
			continue
		}

		progress, ok := stats[badge.Condition]
		if !ok {
			continue
		}

		if progress >= int64(badge.Threshold) {
			// Award badge
			userBadge := models.UserBadge{
				UserID:     userID,
				BadgeID:    badge.ID,
				Progress:   int(progress),
				UnlockedAt: time.Now(),
			}

			if err := database.DB.Create(&userBadge).Error; err == nil {
				newBadges = append(newBadges, badge)
			}
		}
	}

	return newBadges, nil
}
