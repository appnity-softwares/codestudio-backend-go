package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityType string

const (
	ActivityNewSnippet  ActivityType = "NEW_SNIPPET"
	ActivityFollow      ActivityType = "FOLLOW"
	ActivityLike        ActivityType = "LIKE"
	ActivityComment     ActivityType = "COMMENT"
	ActivityAchievement ActivityType = "ACHIEVEMENT"
	ActivityNewUser     ActivityType = "NEW_USER"
	ActivityFork        ActivityType = "FORK"
)

type UserActivity struct {
	ID        string       `gorm:"primaryKey;type:text" json:"id"`
	Type      ActivityType `gorm:"type:text;not null" json:"type"`
	ActorID   string       `gorm:"index;not null" json:"actorId"`
	Actor     User         `gorm:"foreignKey:ActorID" json:"actor"`
	TargetID  string       `gorm:"index" json:"targetId"` // Snippet ID, User ID, etc.
	Message   string       `json:"message"`
	CreatedAt time.Time    `json:"createdAt"`
}

func (UserActivity) TableName() string {
	return "user_activities"
}

func (ua *UserActivity) BeforeCreate(tx *gorm.DB) (err error) {
	if ua.ID == "" {
		ua.ID = uuid.New().String()
	}
	if ua.CreatedAt.IsZero() {
		ua.CreatedAt = time.Now()
	}
	return
}
