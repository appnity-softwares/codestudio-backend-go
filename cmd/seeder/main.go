package main

import (
	"log"

	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/internal/seeds"
)

func main() {
	config.LoadConfig()
	database.Connect()

	log.Println("🚀 Starting Production Data Seeding (Additive & Safe)...")

	// 1. Ensure Migrations (Structure)
	log.Println("🔄 Verifying Schema...")
	database.DB.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.Snippet{},
		&models.Submission{},
		&models.Registration{},
		&models.Problem{},
		&models.ChangelogEntry{},
		&models.PracticeProblem{},
		&models.PracticeSubmission{},
		&models.AdminAction{},
		&models.UserSuspension{},
		&models.TrustScoreHistory{},
		&models.SystemSettings{},
		&models.AdminAuditLog{},
		&models.SubmissionFlag{},
		&models.SubmissionMetrics{},
		&models.TestCase{},
		&models.FeedbackMessage{},
		&models.FeedbackReaction{},
		&models.FeedbackDisagree{},
		&models.Badge{},
		&models.UserBadge{},
	)

	// 2. SYSTEM USER (The owner of all official content)
	systemUser, err := seeds.GetOrCreateSystemUser()
	if err != nil {
		log.Fatalf("❌ Failed to get/create system user: %v", err)
	}

	// 3. SEED CONTENT
	seeds.SeedOfficialSnippets(systemUser)
	seeds.SeedPracticeProblems(systemUser)
	seeds.SeedOfficialContests(systemUser)
	seeds.SeedFeedback(systemUser)
	seeds.SeedBadges()
	seeds.SeedChangelog()

	log.Println("✅ Production Seeding Completed Successfully!")
}
