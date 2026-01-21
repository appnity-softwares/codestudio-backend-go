package seeds

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func SeedUsers() (models.User, error) {
	log.Println("👤 Seeding Users...")

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	admin := models.User{
		ID:       uuid.New().String(),
		Username: "admin",
		Email:    "admin@appnity.cloud",
		Password: string(hash),
		Role:     "ADMIN",
		Image:    "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
	}

	// Check if admin exists
	var existingAdmin models.User
	if err := database.DB.Where("username = ?", "admin").First(&existingAdmin).Error; err != nil {
		if err := database.DB.Create(&admin).Error; err != nil {
			return models.User{}, err
		}
		log.Printf("   ✅ Admin User Created: %s", admin.Username)
	} else {
		admin = existingAdmin
		log.Printf("   ℹ️ Admin User already exists: %s", admin.Username)
	}

	// Create 10 regular users for interaction
	for i := 1; i <= 10; i++ {
		username := fmt.Sprintf("dev_user_%d", i)
		email := fmt.Sprintf("user%d@appnity.cloud", i)

		u := models.User{
			ID:       uuid.New().String(),
			Username: username,
			Email:    email,
			Password: string(hash),
			Role:     "USER",
			Image:    fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", username),
		}

		var existing models.User
		if err := database.DB.Where("username = ?", u.Username).First(&existing).Error; err != nil {
			database.DB.Create(&u)
			log.Printf("   ✅ User Created: %s", u.Username)
		}
	}

	return admin, nil
}
