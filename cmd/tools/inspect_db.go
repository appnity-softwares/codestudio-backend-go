package main

import (
	"fmt"

	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/pushp314/devconnect-backend/internal/database"
)

func main() {
	config.LoadConfig()
	database.Connect()

	var cols []string
	database.DB.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = 'Snippet'").Scan(&cols)
	fmt.Println("Columns:", cols)
}
