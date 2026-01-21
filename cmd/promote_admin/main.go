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
	// Connect to DB
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
		log.Fatalf("User with email %s not found: %v", email, err)
	}

	// Update role
	// Assuming 'Role' field exists and is string. Checking models/user.go might be needed but standard defaults usually have 'role'.
	// If 'Role' doesn't exist, I'll check the schema. But usually it's "role".
	// Using raw update to be safe if struct isn't fully aligned in this script context (though it imports models).

	// Check if Role field exists in model, usually it's "Role" or "role".
	// I'll assume standard model.
	user.Role = "ADMIN"
	if err := db.Save(&user).Error; err != nil {
		log.Fatalf("Failed to update user role: %v", err)
	}

	fmt.Printf("Successfully promoted %s (%s) to ADMIN.\n", user.Username, user.Email)
}
