package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=devconnect port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	email := "appnitysoftwares@gmail.com"
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		log.Fatalf("User %s not found: %v", email, err)
	}
	fmt.Printf("User Found: %s (ID: %s)\n", user.Username, user.ID)

	// Check snippets using raw SQL to inspect columns
	var count1, count2 int64

	// Check 'authorId' (quoted)
	db.Raw("SELECT count(*) FROM \"Snippet\" WHERE \"authorId\" = ?", user.ID).Scan(&count1)
	fmt.Printf("Count matching \"authorId\": %d\n", count1)

	// Check 'author_id' (legacy snake_case)
	db.Raw("SELECT count(*) FROM \"Snippet\" WHERE author_id = ?", user.ID).Scan(&count2)
	fmt.Printf("Count matching author_id: %d\n", count2)

	// Check without quotes if possible (case insensitive matching?)
	// db.Raw("SELECT count(*) FROM \"Snippet\" WHERE authorid = ?", user.ID).Scan(&count3)

	// Dump actual column values for debugging
	var snippets []map[string]interface{}
	db.Raw("SELECT id, \"authorId\", author_id, title FROM \"Snippet\" LIMIT 5").Scan(&snippets)
	fmt.Println("Sample Snippets Data:")
	for _, s := range snippets {
		fmt.Printf("ID: %v | authorId: %v | author_id: %v | Title: %v\n", s["id"], s["authorId"], s["author_id"], s["title"])
	}
}
