package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationTypeLike    NotificationType = "LIKE"
	NotificationTypeComment NotificationType = "COMMENT"

	NotificationTypeSystem      NotificationType = "SYSTEM"
	NotificationTypeLinkRequest NotificationType = "LINK_REQUEST"
	NotificationTypeLinkAccept  NotificationType = "LINK_ACCEPT"
)

type Notification struct {
	ID        string           `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	UserID    string           `gorm:"index;type:text;not null" json:"userId"` // Recipient
	ActorID   string           `gorm:"index;type:text" json:"actorId"`         // Who performed action
	Type      NotificationType `gorm:"type:varchar(20);not null" json:"type"`
	SnippetID *string          `gorm:"index;type:text" json:"snippetId,omitempty"`
	CommentID *string          `gorm:"index;type:text" json:"commentId,omitempty"`
	Message   string           `gorm:"type:text" json:"message"`
	IsRead    bool             `gorm:"default:false" json:"isRead"`
	CreatedAt time.Time        `json:"createdAt"`

	// Relations
	User    User     `gorm:"foreignKey:UserID" json:"-"`
	Actor   User     `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
	Snippet *Snippet `gorm:"foreignKey:SnippetID" json:"snippet,omitempty"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	return
}
