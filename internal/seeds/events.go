package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedOfficialContests(systemUser models.User) {
	log.Println("🌱 Seeding Official Contests...")

	// 1. Upcoming / Active Contest
	// Name: "CodeStudio Launch Challenge"
	// Duration: 2 hours
	// Problems: 3 (Easy–Medium)
	activeContest := models.Event{
		Title:       "CodeStudio Launch Challenge",
		Description: "Welcome to our launch event! Solve 3 problems to earn badges + leaderboard recognition. \n\n**Prizes:**\n- Top 10: Early Adopter Badge\n- Top 3: Profile Spotlight",
		Slug:        "codestudio-launch-challenge", // URL-friendly slug
		Banner:      "https://images.unsplash.com/photo-1504384308090-c54be3855833?q=80&w=2662&auto=format&fit=crop",
		StartTime:   time.Now().Add(10 * time.Minute), // Starts soon/active
		EndTime:     time.Now().Add(2 * time.Hour),
		Price:       0,
		Status:      models.EventStatusLive,
		CreatedBy:   systemUser.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createOrGetEvent(&activeContest)

	// Add 3 Problems (Easy/Medium)
	seedContestProblem(activeContest.ID, "Array Balancing", "Make sum of left and right equal.", "EASY", 100)
	seedContestProblem(activeContest.ID, "String Permutations", "Find all distinct permutations.", "MEDIUM", 200)
	seedContestProblem(activeContest.ID, "Grid Traversal", "Find shortest path in grid with obstacles.", "MEDIUM", 300)

	// 2. Past Contest (Completed)
	// Name: "Getting Started Contest"
	// Status: ENDED
	pastContest := models.Event{
		Title:       "Getting Started Contest",
		Description: "A sample contest to demonstrate the leaderboard.",
		Slug:        "getting-started-contest",
		Banner:      "https://images.unsplash.com/photo-1451187580459-43490279c0fa?q=80&w=2672&auto=format&fit=crop",
		StartTime:   time.Now().Add(-5 * time.Hour),
		EndTime:     time.Now().Add(-3 * time.Hour), // Ended 3 hours ago
		Price:       0,
		Status:      models.EventStatusEnded,
		CreatedBy:   systemUser.ID,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	createOrGetEvent(&pastContest)
	// Add problems if needed for view logic (optional but good for completeness)
	seedContestProblem(pastContest.ID, "Simple Addition", "Add two numbers.", "EASY", 50)
	seedContestProblem(pastContest.ID, "Loop Basics", "Print 1 to N.", "EASY", 50)
}

func createOrGetEvent(evt *models.Event) {
	var existing models.Event
	if err := database.DB.Where("slug = ?", evt.Slug).First(&existing).Error; err == nil {
		log.Printf("   ℹ️ Contest already exists: %s", evt.Title)
		*evt = existing // Update reference to allow problem seeding using correct ID
		return
	}

	evt.ID = uuid.New().String()
	if err := database.DB.Create(evt).Error; err != nil {
		log.Fatalf("❌ Failed to create event %s: %v", evt.Title, err)
	}
	log.Printf("   🏆 Contest Created: %s", evt.Title)
}

func seedContestProblem(eventID, title, description, difficulty string, points int) {
	// Check exist
	var existing models.Problem
	if err := database.DB.Where("event_id = ? AND title = ?", eventID, title).First(&existing).Error; err == nil {
		return
	}

	prob := models.Problem{
		ID:          uuid.New().String(),
		EventID:     eventID,
		Title:       title,
		Description: description,
		Difficulty:  difficulty,
		Points:      points,
		TimeLimit:   2.0,
		MemoryLimit: 128,
		Order:       1, // Order logic normally needs incrementing
		StarterCode: "def solve():\n    pass",
		TestCases: []models.TestCase{
			{ID: uuid.New().String(), Input: "1 2", Output: "3", IsHidden: false},
		},
	}
	// Assign ProblemID to test cases (GORM usually handles this but good to be explicit for struct)
	for i := range prob.TestCases {
		prob.TestCases[i].ProblemID = prob.ID
	}

	if err := database.DB.Create(&prob).Error; err != nil {
		log.Printf("   ❌ Failed: %s - %v", title, err)
	} else {
		log.Printf("      🧩 Problem Added: %s", title)
	}
}
