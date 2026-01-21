package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Load Config & Connect Database
	config.LoadConfig()
	database.Connect()

	fmt.Println("⚠️  WARNING: This will PERMANENTLY DELETE all data in the database.")
	fmt.Println("Proceeding in 3 seconds...")
	time.Sleep(3 * time.Second)

	// 2. Clear All Tables
	clearDatabase()

	// 3. Seed Users
	adminID := "appnity-admin-id"
	emails := []string{
		"pusprajsharma314@gmail.com",
		"jsaurabh334@gmail.com",
		"sausha314@gmail.com",
		"kamalabai181@gmail.com",
		"appnitysoftware@gmail.com", // Admin
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("CodeStudio@2026"), bcrypt.DefaultCost)
	users := make([]models.User, 0)

	for _, email := range emails {
		role := models.RoleUser
		id := uuid.New().String()
		username := ""

		switch email {
		case "pusprajsharma314@gmail.com":
			username = "puspraj"
			role = models.RoleAdmin // Making Puspraj Admin
		case "jsaurabh334@gmail.com":
			username = "saurabh"
		case "sausha314@gmail.com":
			username = "sausha"
		case "kamalabai181@gmail.com":
			username = "kamala"
		case "appnitysoftware@gmail.com":
			username = "appnity"
			role = models.RoleAdmin
			id = adminID // Keep a fixed ID for simplicity in seeding relations
		}

		user := models.User{
			ID:                  id,
			Email:               email,
			Username:            username,
			Name:                username,
			Password:            string(hashedPassword),
			Role:                role,
			OnboardingCompleted: true,
			TrustScore:          100,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		if err := database.DB.Create(&user).Error; err != nil {
			log.Fatalf("Failed to seed user %s: %v", email, err)
		}
		users = append(users, user)
		fmt.Printf("✅ Seeded user: %s\n", email)
	}

	// 4. Seed Snippets (5 per user)
	for _, user := range users {
		for i := 1; i <= 5; i++ {
			snippet := models.Snippet{
				ID:                  uuid.New().String(),
				Title:               fmt.Sprintf("%s's Snippet #%d", user.Username, i),
				Description:         fmt.Sprintf("A cool snippet by %s tracking achievement %d", user.Username, i),
				Code:                "print('Hello CodeStudio!')",
				Language:            "python",
				Status:              "PUBLISHED",
				Verified:            true,
				LastExecutionStatus: "SUCCESS", // REQUIRED to pass BeforeSave validation
				AuthorID:            user.ID,
				Visibility:          "public",
				ExecutionLanguage:   "python",
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			}
			if err := database.DB.Create(&snippet).Error; err != nil {
				log.Printf("❌ Failed to seed snippet for %s: %v", user.Username, err)
			}
		}
	}
	fmt.Println("✅ Snippet Seeding Attempt Complete")

	// 5. Seed 24 Challenges (Events & Problems)
	for i := 1; i <= 24; i++ {
		eventID := uuid.New().String()
		startTime := time.Now().Add(time.Duration(i) * 24 * time.Hour)
		event := models.Event{
			ID:          eventID,
			Title:       fmt.Sprintf("CodeStudio Sprint #%d", i),
			Description: "An intense 2-hour coding sprint to test your algorithms.",
			Slug:        fmt.Sprintf("sprint-%d", i),
			Status:      models.EventStatusUpcoming,
			StartTime:   startTime,
			EndTime:     startTime.Add(2 * time.Hour),
			CreatedBy:   adminID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if result := database.DB.Create(&event); result.Error != nil {
			log.Printf("Failed to seed event %d: %v", i, result.Error)
			continue
		}

		// Add 1 problem per challenge
		problem := models.Problem{
			ID:          uuid.New().String(),
			EventID:     eventID,
			Title:       fmt.Sprintf("Algorithmic Mystery #%d", i),
			Description: "Implement a function that solves the mystery of the universe.",
			Difficulty:  "MEDIUM",
			Points:      100,
			Order:       1,
			StarterCode: "def solve():\n    pass",
		}
		database.DB.Create(&problem)
	}
	fmt.Println("✅ Seeded 24 Challenges")

	// 6. Seed 1 Practice Arena with 4 problems
	for i := 1; i <= 4; i++ {
		practiceProb := models.PracticeProblem{
			ID:           uuid.New().String(),
			Title:        fmt.Sprintf("Practice Discovery #%d", i),
			Description:  "Get comfortable with the platform by solving this guided problem.",
			Difficulty:   "EASY",
			Category:     "BASICS",
			StarterCode:  "def solution():\n    # Write your code here\n    pass",
			SolutionCode: "def solution():\n    print(\"Hello World\")",
			TestCases:    "[{\"input\": \"\", \"expected\": \"Hello World\\n\"}]",
			Language:     "python",
			CreatorID:    adminID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		database.DB.Create(&practiceProb)
	}
	fmt.Println("✅ Seeded 4 Practice Problems")

	// 7. Feedback Wall - Empty with Admin Message
	adminMsg := models.FeedbackMessage{
		ID:        uuid.New().String(),
		UserID:    adminID,
		Content:   "Welcome to the CodeStudio Feedback Wall! 🚀 Share your ideas and report bugs here. Let's build together.",
		Category:  models.CategoryOther,
		Status:    models.StatusOpen,
		IsPinned:  true,
		CreatedAt: time.Now(),
	}
	database.DB.Create(&adminMsg)
	fmt.Println("✅ Seeded Admin Feedback Message")

	// 8. Changelogs
	changelog := models.ChangelogEntry{
		ID:          uuid.New().String(),
		Version:     "v2.0.0",
		Title:       "The New Era: Landing Page & Performance",
		Description: "Introducing the all-new Landing Page and massive performance improvements across the Platform.",
		ReleaseType: "FEATURE",
		IsPublished: true,
		ReleasedAt:  func() *time.Time { t := time.Now(); return &t }(),
		CreatedBy:   adminID,
		CreatedAt:   time.Now(),
	}
	database.DB.Create(&changelog)
	fmt.Println("✅ Seeded Initial Changelog")

	fmt.Println("\n✨ Production Reset & Seeding Complete!")
}

func clearDatabase() {
	fmt.Println("🗑️  Cleaning database...")

	// List of tables to truncate
	tables := []string{
		"feedback_reactions",
		"feedback_disagrees",
		"feedback_messages",
		"submission_flags",
		"submission_metrics",
		"submissions",
		"registrations",
		"problems",
		"test_cases",
		"events",
		"Snippet",
		"avatar_seeds",
		"role_permissions",
		"practice_submissions",
		"practice_problems",
		"admin_actions",
		"user_suspensions",
		"trust_score_histories",
		"system_settings",
		"admin_audit_logs",
		"entity_views",
		"entity_copies",
		"changelog_entries",
		"messages",
		"User", // Truncate User last
	}

	for _, table := range tables {
		if err := database.DB.Exec(fmt.Sprintf("TRUNCATE TABLE \"%s\" CASCADE", table)).Error; err != nil {
			// Some tables might not have quotes or might have different case
			// Attempt lowercase version if quoted fails
			database.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		}
	}

	fmt.Println("✅ Database cleaned")
}
