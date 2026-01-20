package database

import (
	"log"
	"time"

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

	// Configure connection pool for production performance
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// Production-grade connection pool settings
	sqlDB.SetMaxOpenConns(25)                 // Max open connections to DB
	sqlDB.SetMaxIdleConns(10)                 // Max idle connections in pool
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Connection max lifetime

	DB = db
	log.Println("Connected to PostgreSQL with connection pooling (max: 25, idle: 10)")
}
