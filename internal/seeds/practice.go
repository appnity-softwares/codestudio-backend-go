package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedPracticeProblems(creator models.User) {
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
	}

	for _, p := range problems {
		if err := database.DB.Create(&p).Error; err != nil {
			log.Printf("   ❌ Failed: %s - %v", p.Title, err)
		} else {
			log.Printf("   🧩 Practice Problem Added: %s (%s)", p.Title, p.Difficulty)
		}
	}
}
