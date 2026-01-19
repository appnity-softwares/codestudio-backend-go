package database

import (
	"log"

	"github.com/pushp314/devconnect-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := config.AppConfig.DatabaseURL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB = db
	log.Println("Connected to PostgreSQL database successfully")

	// Auto-migrate Activity table (Removed for MVP)
}
