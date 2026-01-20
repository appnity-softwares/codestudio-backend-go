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

	log.Println("🔄 Running migrations (Stage 1: Tables)...")
	// Temporarily disable foreign key constraints to break circular dependency (User <-> Snippet)
	database.DB.Config.DisableForeignKeyConstraintWhenMigrating = true

	migrationModels := []interface{}{
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
	}

	for _, m := range migrationModels {
		if err := database.DB.AutoMigrate(m); err != nil {
			log.Fatalf("❌ Failed to migrate table for %T: %v", m, err)
		}
	}

	log.Println("🔄 Running migrations (Stage 2: Constraints)...")
	// Re-enable and add all foreign key constraints via ALTER TABLE
	database.DB.Config.DisableForeignKeyConstraintWhenMigrating = false
	if err := database.DB.AutoMigrate(migrationModels...); err != nil {
		log.Fatalf("❌ Failed to add constraints: %v", err)
	}

	log.Println("🗑️  Clearing Tables (EXCEPT Users)...")
	// Note: We use CASCADE to clean up related data
	// We do NOT truncate "User" table to preserve admin accounts.
	tablesToTruncate := []string{
		"\"Snippet\"", "events", "problems", "submissions",
		"registrations", "changelog_entries",
		"practice_problems", "practice_submissions",
		"submission_flags", "submission_metrics", "test_cases",
	}

	for _, table := range tablesToTruncate {
		if err := database.DB.Exec("TRUNCATE TABLE \"" + table + "\" RESTART IDENTITY CASCADE").Error; err != nil {
			log.Printf("⚠️ Warning: Failed to truncate %s: %v", table, err)
		}
	}

	// 👤 SEED USERS (and get admin for ownership)
	admin, err := seeds.SeedUsers()
	if err != nil {
		log.Fatalf("❌ Failed to seed users: %v", err)
	}

	// 🌱 RUN MODULAR SEEDERS
	seeds.SeedEvents(admin)
	seeds.SeedPracticeProblems(admin)
	seeds.SeedSnippets(admin)
	seeds.SeedChangelog()

	log.Println("✅ Database Reset & Seeding Complete!")
}
