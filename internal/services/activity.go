package services

import (
	"log"
	"time"

	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func LogActivity(actorID string, activityType models.ActivityType, targetID string, message string) {
	activity := models.UserActivity{
		Type:      activityType,
		ActorID:   actorID,
		TargetID:  targetID,
		Message:   message,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&activity).Error; err != nil {
		log.Printf("Failed to log activity: %v", err)
	}
}
