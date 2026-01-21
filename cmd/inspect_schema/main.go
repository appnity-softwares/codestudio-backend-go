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
	// Connect to DB (using env vars or default)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=devconnect port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Check table name
	stmt := &gorm.Statement{DB: db}
	stmt.Parse(&models.Snippet{})
	tableName := stmt.Schema.Table
	fmt.Println("Table Name:", tableName)

	// List columns from information schema
	var columns []string
	db.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = ?", tableName).Scan(&columns)

	fmt.Println("Columns in table", tableName, ":")
	for _, col := range columns {
		fmt.Println("-", col)
	}

	// Also try "snippet" and "snippets" just in case
	db.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = ?", "snippet").Scan(&columns)
	if len(columns) > 0 {
		fmt.Println("Columns in table 'snippet':")
		for _, col := range columns {
			fmt.Println("-", col)
		}
	}
}
