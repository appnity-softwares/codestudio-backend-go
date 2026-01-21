package seeds

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func GetOrCreateSystemUser() (models.User, error) {
	log.Println("👤 Checking System User...")

	username := "codestudio"
	email := "official@codestudio.dev"

	var user models.User
	err := database.DB.Where("username = ?", username).First(&user).Error

	if err == nil {
		log.Printf("   ✅ System User found: %s", user.Username)
		return user, nil
	}

	// Create if not exists
	hash, _ := bcrypt.GenerateFromPassword([]byte("CodeStudioOfficial2024!"), bcrypt.DefaultCost)

	user = models.User{
		ID:        uuid.New().String(),
		Username:  username,
		Email:     email,
		Password:  string(hash),
		Role:      models.RoleAdmin,
		Name:      "CodeStudio Team",
		Bio:       "Official CodeStudio account. Announcements, examples, and contests.",
		Image:     "https://api.dicebear.com/7.x/identicon/svg?seed=codestudio",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return models.User{}, err
	}

	log.Printf("   ✅ System User Created: %s", user.Username)
	return user, nil
}
