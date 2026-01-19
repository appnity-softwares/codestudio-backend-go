package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	config.LoadConfig()
	database.Connect()

	log.Println("🔄 Running migrations (just in case)...")
	database.DB.AutoMigrate(
		&models.User{},
		&models.Snippet{},
		&models.Event{},
		&models.Submission{},
		&models.Registration{},
		&models.Problem{},
		&models.ChangelogEntry{},
		&models.PracticeProblem{},
		&models.PracticeSubmission{},
		// Admin models
		&models.AdminAction{},
		&models.UserSuspension{},
		&models.TrustScoreHistory{},
		&models.SystemSettings{},
		&models.AdminAuditLog{},
		&models.SubmissionFlag{},
		&models.SubmissionMetrics{},
		&models.TestCase{},
	)

	log.Println("🗑️  Clearing Tables (EXCEPT Users)...")
	// Note: We use CASCADE to clean up related data
	// We do NOT truncate "User" table.
	if err := database.DB.Exec("TRUNCATE TABLE \"Snippet\", events, problems, submissions, registrations, changelog_entries, practice_problems, practice_submissions RESTART IDENTITY CASCADE").Error; err != nil {
		log.Fatalf("❌ Failed to truncate: %v", err)
	}

	log.Println("👤 Fetching Admin User...")
	var admin models.User
	if err := database.DB.Where("role = ?", "ADMIN").First(&admin).Error; err != nil {
		log.Println("⚠️ No ADMIN found. Fetching ANY user to be the creator...")
		if err := database.DB.First(&admin).Error; err != nil {
			log.Println("⚠️ No users found at all! Creating a fallback admin...")
			hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			admin = models.User{
				ID:       uuid.New().String(),
				Username: "admin",
				Email:    "admin@devconnect.com",
				Password: string(hash),
				Role:     "ADMIN",
				Image:    "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
			}
			database.DB.Create(&admin)
		}
	}
	log.Printf("👉 Using Creator: %s (%s)", admin.Username, admin.ID)

	seedOfficialContests(admin)
	seedPracticeArena(admin)
	seedSnippets(admin)
	seedChangelog()
	seedPracticeProblems(admin) // v1.2: Practice Arena Problems

	log.Println("✅ Database Reset & Seeding Complete!")
}

func seedChangelog() {
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
			Version:   "1.1.3",
			Title:     "Stability Patch",
			Changes:   []string{"Fixed socket auth", "Improved contest entry flow", "Enhanced seed data"},
			CreatedAt: time.Now().Add(-24 * time.Hour),
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

func seedSnippets(creator models.User) {
	log.Println("📜 Seeding Code Snippets...")

	snippets := []models.Snippet{
		{
			ID:                  uuid.New().String(),
			Title:               "Depth First Search",
			Description:         "A standard DFS implementation in Python",
			Code:                "def dfs(node, visited):\n    if node in visited:\n        return\n    visited.add(node)\n    print(f'Visiting {node}')\n    for neighbor in [node+1]: # Dummy neighbors\n        if neighbor < 5: dfs(neighbor, visited)\n\nvisited = set()\ndfs(0, visited)",
			Language:            "python",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			OutputSnapshot:      `{"run":{"stdout":"Visiting 0\nVisiting 1\nVisiting 2\nVisiting 3\nVisiting 4\n","stderr":"","code":0}}`,
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "TERMINAL",
		},
		{
			ID:          uuid.New().String(),
			Title:       "Modern Neumorphic Card",
			Description: "A beautiful UI component using HTML/CSS",
			Code: `<div style="padding: 40px; background: #e0e0e0; border-radius: 50px; box-shadow: 20px 20px 60px #bebebe, -20px -20px 60px #ffffff; color: #444; text-align: center; font-family: sans-serif;">
  <h1 style="margin-bottom: 20px;">Neumorphism</h1>
  <p>Soft UI is the new trend.</p>
  <button style="margin-top: 20px; padding: 12px 24px; border-radius: 12px; background: #e0e0e0; border: none; box-shadow: 5px 5px 10px #bebebe, -5px -5px 10px #ffffff; cursor: pointer; color: #666; font-weight: bold;">Click Me</button>
</div>`,
			Language:            "html",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "WEB_PREVIEW",
		},
		{
			ID:          uuid.New().String(),
			Title:       "Glassmorphism Counter",
			Description: "Interactive React component with glassmorphism",
			Code: `import React, { useState } from 'react';

export default function Counter() {
  const [count, setCount] = useState(0);

  return (
    <div style={{
      padding: '40px',
      background: 'rgba(255, 255, 255, 0.1)',
      backdropFilter: 'blur(10px)',
      borderRadius: '24px',
      border: '1px solid rgba(255, 255, 255, 0.2)',
      color: 'white',
      textAlign: 'center',
      fontFamily: 'system-ui'
    }}>
      <h2 style={{ margin: '0 0 20px 0' }}>React Glass Counter</h2>
      <div style={{ fontSize: '48px', fontWeight: 'bold', margin: '20px 0' }}>{count}</div>
      <button 
        onClick={() => setCount(c => c + 1)}
        style={{
          padding: '10px 20px',
          borderRadius: '12px',
          background: '#6366f1',
          color: 'white',
          border: 'none',
          cursor: 'pointer',
          fontWeight: 'bold'
        }}
      >
        Increment
      </button>
    </div>
  );
}`,
			Language:            "react",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "WEB_PREVIEW",
		},
		{
			ID:                  uuid.New().String(),
			Title:               "Fibonacci Sequence",
			Description:         "Efficient recursive approach with memoization",
			Code:                "function fib(n, memo = {}) {\n  if (n in memo) return memo[n];\n  if (n <= 2) return 1;\n  memo[n] = fib(n - 1, memo) + fib(n - 2, memo);\n  return memo[n];\n}\n\nconsole.log(fib(10));",
			Language:            "typescript",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			OutputSnapshot:      `{"run":{"stdout":"55\n","stderr":"","code":0}}`,
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "TERMINAL",
		},
		{
			ID:                  uuid.New().String(),
			Title:               "Stream API",
			Description:         "Java 8+ Streams",
			Code:                "import java.util.*;\npublic class Main {\n    public static void main(String[] args) {\n        List<String> list = Arrays.asList(\"a\", \"b\", \"c\");\n        list.stream().map(String::toUpperCase).forEach(System.out::println);\n    }\n}",
			Language:            "java",
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			OutputSnapshot:      `{"run":{"stdout":"A\nB\nC\n","stderr":"","code":0}}`,
			Visibility:          "PUBLIC",
			AuthorID:            creator.ID,
			PreviewType:         "TERMINAL",
		},
	}

	for _, s := range snippets {
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
		if err := database.DB.Create(&s).Error; err != nil {
			log.Printf("❌ Failed to create snippet %s: %v", s.Title, err)
		} else {
			log.Printf("   📝 Snippet Added: %s (%s)", s.Title, s.Language)
		}
	}
}

func seedOfficialContests(creator models.User) {
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
}

func seedPracticeArena(creator models.User) {
	log.Println("🌱 Seeding Practice Arena...")
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
		log.Printf("⚠️ Practice Arena might already exist, skipping creation.")
	} else {
		log.Printf("🏟️ Practice Arena Created")
		seedProblem(practiceArena.ID, "Two Sum", "Find indices of two numbers that add up to target.", "Easy")
	}
}

func createEvent(evt models.Event) {
	if err := database.DB.Create(&evt).Error; err != nil {
		log.Fatalf("❌ Failed to create event %s: %v", evt.Title, err)
	}
	log.Printf("🏆 Event Created: %s (Start: %s)", evt.Title, evt.StartTime.Format(time.RFC822))
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
		log.Printf("❌ Failed to create problem %s: %v", title, err)
	} else {
		log.Printf("   🧩 Problem Added: %s", title)
	}
}

// v1.2: Seed Practice Arena Problems
func seedPracticeProblems(creator models.User) {
	log.Println("🏋️ Seeding Practice Problems...")

	problems := []models.PracticeProblem{
		{
			ID:             uuid.New().String(),
			Title:          "Hello World",
			Description:    "Write a program that prints 'Hello, World!' to the console.\n\nThis is the classic first program for any language.",
			Difficulty:     "EASY",
			Category:       "Basics",
			StarterCode:    "# Write your solution here\n",
			TestCases:      `[{"input": "", "expected": "Hello, World!"}]`,
			Language:       "python",
			TimeLimit:      2,
			MemoryLimit:    128,
			IsDailyProblem: true,
			CreatorID:      creator.ID,
			CreatedAt:      time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Title:       "Sum of Two Numbers",
			Description: "Given two integers a and b, return their sum.\n\nExample:\nInput: a = 5, b = 3\nOutput: 8",
			Difficulty:  "EASY",
			Category:    "Math",
			StarterCode: "function sum(a, b) {\n  // Your code here\n}",
			TestCases:   `[{"input": "5 3", "expected": "8"}, {"input": "0 0", "expected": "0"}]`,
			Language:    "javascript",
			TimeLimit:   2,
			MemoryLimit: 128,
			CreatorID:   creator.ID,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Title:       "Reverse a String",
			Description: "Write a function that reverses a string.\n\nExample:\nInput: 'hello'\nOutput: 'olleh'",
			Difficulty:  "EASY",
			Category:    "Strings",
			StarterCode: "def reverse_string(s):\n    # Your code here\n    pass",
			TestCases:   `[{"input": "hello", "expected": "olleh"}, {"input": "world", "expected": "dlrow"}]`,
			Language:    "python",
			TimeLimit:   2,
			MemoryLimit: 128,
			CreatorID:   creator.ID,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Title:       "FizzBuzz",
			Description: "Print numbers from 1 to n. For multiples of 3, print 'Fizz'. For multiples of 5, print 'Buzz'. For multiples of both, print 'FizzBuzz'.\n\nExample: n=15 should output:\n1, 2, Fizz, 4, Buzz, Fizz, 7, 8, Fizz, Buzz, 11, Fizz, 13, 14, FizzBuzz",
			Difficulty:  "MEDIUM",
			Category:    "Logic",
			StarterCode: "function fizzBuzz(n) {\n  // Your code here\n}",
			TestCases:   `[{"input": "15", "expected": "1\n2\nFizz\n4\nBuzz\nFizz\n7\n8\nFizz\nBuzz\n11\nFizz\n13\n14\nFizzBuzz"}]`,
			Language:    "javascript",
			TimeLimit:   2,
			MemoryLimit: 128,
			CreatorID:   creator.ID,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Title:       "Palindrome Check",
			Description: "Write a function that checks if a given string is a palindrome (reads the same forwards and backwards).\n\nExample:\nInput: 'racecar' → true\nInput: 'hello' → false",
			Difficulty:  "MEDIUM",
			Category:    "Strings",
			StarterCode: "func isPalindrome(s string) bool {\n    // Your code here\n    return false\n}",
			TestCases:   `[{"input": "racecar", "expected": "true"}, {"input": "hello", "expected": "false"}]`,
			Language:    "go",
			TimeLimit:   2,
			MemoryLimit: 128,
			CreatorID:   creator.ID,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Title:       "Find Maximum in Array",
			Description: "Given an array of integers, find the maximum value.\n\nExample:\nInput: [3, 1, 4, 1, 5, 9, 2, 6]\nOutput: 9",
			Difficulty:  "EASY",
			Category:    "Arrays",
			StarterCode: "def find_max(arr):\n    # Your code here\n    pass",
			TestCases:   `[{"input": "[3,1,4,1,5,9,2,6]", "expected": "9"}]`,
			Language:    "python",
			TimeLimit:   2,
			MemoryLimit: 128,
			CreatorID:   creator.ID,
			CreatedAt:   time.Now(),
		},
	}

	for _, p := range problems {
		if err := database.DB.Create(&p).Error; err != nil {
			log.Printf("   ❌ Failed: %s - %v", p.Title, err)
		} else {
			log.Printf("   🧩 Practice Problem Added: %s (%s)", p.Title, p.Difficulty)
		}
	}
}
