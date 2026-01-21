package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	database.DB = db

	fmt.Println("--- Practice Submissions ---")
	var submissions []models.PracticeSubmission
	db.Find(&submissions)
	for _, s := range submissions {
		fmt.Printf("User: %s | Problem: %s | Status: %s | Created: %v\n", s.UserID, s.ProblemID, s.Status, s.CreatedAt)
	}

	fmt.Println("\n--- Solved Check Simulation ---")
	if len(submissions) > 0 {
		userID := submissions[0].UserID
		problemID := submissions[0].ProblemID

		var solvedIDs []string
		db.Model(&models.PracticeSubmission{}).
			Where("user_id = ? AND status = ? AND problem_id = ?", userID, "ACCEPTED", problemID).
			Distinct("problem_id").
			Pluck("problem_id", &solvedIDs)

		fmt.Printf("Query for User %s, Problem %s: Found Solved IDs: %v\n", userID, problemID, solvedIDs)
	} else {
		fmt.Println("No submissions found to test.")
	}
}
