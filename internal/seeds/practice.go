package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedPracticeProblems(systemUser models.User) {
	log.Println("🏋️ Seeding Practice Problems...")

	problems := []models.PracticeProblem{
		// EASY (5)
		{Title: "Hello World", Description: "Write a program that prints 'Hello, World!' to the console.", Difficulty: "EASY", Category: "Basics", Language: "python", StarterCode: "print(\"\")"},
		{Title: "Return Sum", Description: "Given two numbers, return their sum.", Difficulty: "EASY", Category: "Basics", Language: "javascript", StarterCode: "function sum(a, b) {\n  return a + b;\n}"},
		{Title: "Is Even?", Description: "Return true if the number is even, else false.", Difficulty: "EASY", Category: "Math", Language: "python", StarterCode: "def is_even(n):\n    return n % 2 == 0"},
		{Title: "Find Max", Description: "Find the maximum number in an array.", Difficulty: "EASY", Category: "Arrays", Language: "python", StarterCode: "def find_max(arr):\n    pass"},
		{Title: "String Length", Description: "Return the length of the given string.", Difficulty: "EASY", Category: "Strings", Language: "javascript", StarterCode: "function getLength(str) {\n  return str.length;\n}"},

		// MEDIUM (5)
		{Title: "FizzBuzz", Description: "Print numbers 1 to n. Mul of 3: Fizz, Mul of 5: Buzz, Both: FizzBuzz.", Difficulty: "MEDIUM", Category: "Logic", Language: "python", StarterCode: "def fizzbuzz(n):\n    pass"},
		{Title: "Palindrome Check", Description: "Check if a string is a palindrome (reads same forwards and backwards).", Difficulty: "MEDIUM", Category: "Strings", Language: "javascript", StarterCode: "function isPalindrome(s) {}"},
		{Title: "Factorial", Description: "Calculate n! (n factorial).", Difficulty: "MEDIUM", Category: "Math", Language: "python", StarterCode: "def factorial(n):\n    pass"},
		{Title: "Prime Number Check", Description: "Check if a number is prime.", Difficulty: "MEDIUM", Category: "Math", Language: "go", StarterCode: "func IsPrime(n int) bool {}"},
		{Title: "Array Deduplication", Description: "Remove duplicate values from an array.", Difficulty: "MEDIUM", Category: "Arrays", Language: "javascript", StarterCode: "function dedupe(arr) {}"},
	}

	for _, p := range problems {
		var existing models.PracticeProblem
		if err := database.DB.Where(&models.PracticeProblem{Title: p.Title, CreatorID: systemUser.ID}).First(&existing).Error; err == nil {
			log.Printf("   ℹ️ Practice Problem already exists: %s", p.Title)
			continue
		}

		newProb := models.PracticeProblem{
			ID:          uuid.New().String(),
			Title:       p.Title,
			Description: p.Description,
			Difficulty:  p.Difficulty,
			Category:    p.Category,
			StarterCode: p.StarterCode,
			Language:    p.Language,
			TimeLimit:   2,
			MemoryLimit: 128,
			CreatorID:   systemUser.ID,
			CreatedAt:   time.Now(),
		}

		if err := database.DB.Create(&newProb).Error; err != nil {
			log.Printf("   ❌ Failed: %s - %v", newProb.Title, err)
		} else {
			log.Printf("   🧩 Practice Problem Added: %s", newProb.Title)
		}
	}
}
