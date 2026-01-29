package integration

import (
	"fmt"
	"testing"

	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	// Using URL format to avoid parsing ambiguities
	baseDSN    = "postgres://pushp314:@localhost:5432/postgres?sslmode=disable"
	testDBName = "devconnect_test"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// 0. Init Config for JWT
	config.AppConfig = &config.Config{
		JWTSecret: "test_secret_key_12345",
	}

	// 1. Connect to default 'postgres' database to create the test DB
	db, err := gorm.Open(postgres.Open(baseDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to postgres DB: %v", err)
	}

	// 2. Drop and Create Test DB
	// Terminate existing connections first to ensure DROP works
	db.Exec(fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'", testDBName))

	if err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName)).Error; err != nil {
		t.Fatalf("Failed to drop test DB: %v", err)
	}

	if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName)).Error; err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}

	// 3. Connect to the new Test DB
	testDSN := fmt.Sprintf("postgres://pushp314:@localhost:5432/%s?sslmode=disable", testDBName)
	testDB, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Error), // Show errors
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}

	// 4. Run Migrations
	// Enable UUID extension
	if err := testDB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		t.Fatalf("Failed to enable uuid-ossp extension: %v", err)
	}

	// Ensure User is created first
	if err := testDB.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("Failed to migrate User table: %v", err)
	}

	err = testDB.AutoMigrate(
		&models.Event{},
		&models.Problem{},
		&models.TestCase{},
		&models.Registration{},
		&models.Submission{},
		&models.SubmissionMetrics{},
		&models.SubmissionFlag{},
		&models.Snippet{},
		&models.UserBadge{},
		&models.Badge{},
		&models.SystemSettings{},
		&models.FeedbackMessage{}, // Added model
	)
	if err != nil {
		t.Fatalf("Failed to migrate test DB: %v", err)
	}

	// 5. Override global DB variable if your handlers use it
	// Ideally handlers should accept DB interface, but for this legacy codebase they utilize global database.DB
	database.DB = testDB

	return testDB
}
