package models

import (
	"time"
)

type FeedbackCategory string

const (
	CategoryBug         FeedbackCategory = "BUG"
	CategoryUX          FeedbackCategory = "UX"
	CategoryFeature     FeedbackCategory = "FEATURE"
	CategoryPerformance FeedbackCategory = "PERFORMANCE"
	CategoryOther       FeedbackCategory = "OTHER"
)

type FeedbackMessage struct {
	ID        string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string           `gorm:"index" json:"userId"`
	User      User             `gorm:"foreignKey:UserID" json:"user"`
	Content   string           `gorm:"type:text;not null" json:"content"`
	Category  FeedbackCategory `gorm:"type:text;default:'OTHER'" json:"category"`
	Upvotes   int              `gorm:"default:0" json:"upvotes"`
	Downvotes int              `gorm:"default:0" json:"downvotes"`
	IsAck     bool             `gorm:"default:false" json:"isAck"` // Acknowledged by team
	CreatedAt time.Time        `json:"createdAt"`

	// Virtual fields for checking if current user reacted
	HasReacted   bool `gorm:"-" json:"hasReacted"`
	HasDisagreed bool `gorm:"-" json:"hasDisagreed"`
}

type FeedbackReaction struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"uniqueIndex:idx_user_message" json:"userId"`
	MessageID string    `gorm:"uniqueIndex:idx_user_message" json:"messageId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (FeedbackMessage) TableName() string {
	return "feedback_messages"
}

func (FeedbackReaction) TableName() string {
	return "feedback_reactions"
}

// FeedbackDisagree tracks downvotes/disagrees on feedback
type FeedbackDisagree struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"uniqueIndex:idx_disagree_user_message" json:"userId"`
	MessageID string    `gorm:"uniqueIndex:idx_disagree_user_message" json:"messageId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (FeedbackDisagree) TableName() string {
	return "feedback_disagrees"
}
