package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedEvents(creator models.User) {
	log.Println("🌱 Seeding Official Contests...")

	// 1. LIVE CONTEST (Started 1 hour ago, Ends in 2 hours)
	liveContest := models.Event{
		ID:          uuid.New().String(),
		Title:       "DevConnect Global Round 1",
		Description: "The official live contest round. Participate now to test your skills!\n\n**Rules:**\n- No cheating.\n- Individual participation only.",
		Slug:        "devconnect-global-round-1",
		Banner:      "https://images.unsplash.com/photo-1504384308090-c54be3855833?q=80&w=2662&auto=format&fit=crop",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now().Add(2 * time.Hour),
		Price:       0,
		Status:      models.EventStatusLive,
		CreatedBy:   creator.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	createEvent(liveContest)
	seedProblem(liveContest.ID, "Matrix Rotation", "Rotate a matrix by 90 degrees clockwise.", "Medium")

	// 2. UPCOMING CONTEST (Starts in 2 hours, Ends in 5 hours)
	upcomingContest := models.Event{
		ID:          uuid.New().String(),
		Title:       "Winter Algorithm Cup 2026",
		Description: "Get ready for the biggest algorithm challenge of the winter. Registration is open.",
		Slug:        "winter-algorithm-cup-2026",
		Banner:      "https://images.unsplash.com/photo-1451187580459-43490279c0fa?q=80&w=2672&auto=format&fit=crop",
		StartTime:   time.Now().Add(2 * time.Hour),
		EndTime:     time.Now().Add(5 * time.Hour),
		Price:       0,
		Status:      models.EventStatusLive, // Status LIVE but StartTime is future = Upcoming
		CreatedBy:   creator.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	createEvent(upcomingContest)
	seedProblem(upcomingContest.ID, "Dynamic Grid Paths", "Find the number of unique paths in a grid with obstacles.", "Hard")

	// 3. Practice Arena
	log.Println("🏟️ Seeding Practice Arena...")
	practiceArena := models.Event{
		ID:          "practice-arena-mvp",
		Title:       "Practice Arena",
		Description: "Sharpen your skills with these practice problems. No time limit.",
		Slug:        "practice-arena",
		Banner:      "https://images.unsplash.com/photo-1517694712202-14dd9538aa97?q=80&w=2940&auto=format&fit=crop",
		StartTime:   time.Now().Add(-8760 * time.Hour),
		EndTime:     time.Now().Add(87600 * time.Hour),
		Price:       0,
		Status:      models.EventStatusLive,
		CreatedBy:   creator.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := database.DB.Create(&practiceArena).Error; err != nil {
		log.Printf("   ⚠️ Practice Arena might already exist, skipping creation.")
	} else {
		log.Printf("   🏟️ Practice Arena Created")
		seedProblem(practiceArena.ID, "Two Sum", "Find indices of two numbers that add up to target.", "Easy")
	}
}

func createEvent(evt models.Event) {
	if err := database.DB.Create(&evt).Error; err != nil {
		log.Fatalf("❌ Failed to create event %s: %v", evt.Title, err)
	}
	log.Printf("   🏆 Event Created: %s (Start: %s)", evt.Title, evt.StartTime.Format(time.RFC822))
}

func seedProblem(eventID, title, description, difficulty string) {
	prob := models.Problem{
		ID:          uuid.New().String(),
		EventID:     eventID,
		Title:       title,
		Description: description,
		Difficulty:  difficulty,
		Points:      100,
		TimeLimit:   2.0,
		MemoryLimit: 128,
		Order:       1,
		StarterCode: "def solve():\n    pass",
		TestCases: []models.TestCase{
			{ID: uuid.New().String(), Input: "1 2", Output: "3", IsHidden: false},
		},
	}
	// Assign ProblemID to test cases
	for i := range prob.TestCases {
		prob.TestCases[i].ProblemID = prob.ID
	}

	if err := database.DB.Create(&prob).Error; err != nil {
		log.Printf("   ❌ Failed to create problem %s: %v", title, err)
	} else {
		log.Printf("      🧩 Problem Added: %s", title)
	}
}
