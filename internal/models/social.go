package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserLink represents a follower/following relationship
type UserLink struct {
	ID        string         `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	LinkerID string `gorm:"uniqueIndex:idx_linker_linked" json:"linkerId"` // The user who follows (Linker)
	Linker   User   `gorm:"foreignKey:LinkerID" json:"linker"`

	LinkedID string `gorm:"uniqueIndex:idx_linker_linked" json:"linkedId"` // The user being followed (Linked)
	Linked   User   `gorm:"foreignKey:LinkedID" json:"linked"`
}

// SnippetLike represents a like on a snippet
type SnippetLike struct {
	ID        string    `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	UserID string `gorm:"uniqueIndex:idx_user_snippet_like" json:"userId"`
	User   User   `gorm:"foreignKey:UserID" json:"user"`

	SnippetID string  `gorm:"uniqueIndex:idx_user_snippet_like" json:"snippetId"`
	Snippet   Snippet `gorm:"foreignKey:SnippetID" json:"snippet"`
}

// SnippetDislike represents a dislike on a snippet
type SnippetDislike struct {
	ID        string    `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	UserID string `gorm:"uniqueIndex:idx_user_snippet_dislike" json:"userId"`
	User   User   `gorm:"foreignKey:UserID" json:"user"`

	SnippetID string  `gorm:"uniqueIndex:idx_user_snippet_dislike" json:"snippetId"`
	Snippet   Snippet `gorm:"foreignKey:SnippetID" json:"snippet"`
}

// Comment represents a comment on a snippet
type Comment struct {
	ID        string         `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Content string `gorm:"type:text" json:"content"`

	UserID string `json:"userId"`
	User   User   `gorm:"foreignKey:UserID" json:"user"`

	SnippetID string  `json:"snippetId"`
	Snippet   Snippet `gorm:"foreignKey:SnippetID" json:"-"`
}

func (UserLink) TableName() string {
	return "UserLink"
}

func (ul *UserLink) BeforeCreate(tx *gorm.DB) (err error) {
	if ul.ID == "" {
		ul.ID = uuid.New().String()
	}
	return
}

func (SnippetLike) TableName() string {
	return "SnippetLike"
}

func (sl *SnippetLike) BeforeCreate(tx *gorm.DB) (err error) {
	if sl.ID == "" {
		sl.ID = uuid.New().String()
	}
	return
}

func (SnippetDislike) TableName() string {
	return "SnippetDislike"
}

func (sd *SnippetDislike) BeforeCreate(tx *gorm.DB) (err error) {
	if sd.ID == "" {
		sd.ID = uuid.New().String()
	}
	return
}

func (Comment) TableName() string {
	return "Comment"
}

func (c *Comment) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return
}

// LinkRequestStatus represents the status of a follow request
type LinkRequestStatus string

const (
	LinkRequestPending  LinkRequestStatus = "PENDING"
	LinkRequestAccepted LinkRequestStatus = "ACCEPTED"
	LinkRequestRejected LinkRequestStatus = "REJECTED"
)

// LinkRequest represents a follow request for private accounts
type LinkRequest struct {
	ID         string            `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	SenderID   string            `gorm:"index" json:"senderId"`
	Sender     User              `gorm:"foreignKey:SenderID" json:"sender"`
	ReceiverID string            `gorm:"index" json:"receiverId"`
	Receiver   User              `gorm:"foreignKey:ReceiverID" json:"receiver"`
	Status     LinkRequestStatus `gorm:"type:text;default:'PENDING'" json:"status"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

func (LinkRequest) TableName() string {
	return "LinkRequest"
}

func (lr *LinkRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if lr.ID == "" {
		lr.ID = uuid.New().String()
	}
	return
}

// UserBlock represents one user blocking another
type UserBlock struct {
	ID        string    `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	BlockerID string    `gorm:"uniqueIndex:idx_blocker_blocked" json:"blockerId"`
	BlockedID string    `gorm:"uniqueIndex:idx_blocker_blocked" json:"blockedId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (UserBlock) TableName() string {
	return "UserBlock"
}

func (ub *UserBlock) BeforeCreate(tx *gorm.DB) (err error) {
	if ub.ID == "" {
		ub.ID = uuid.New().String()
	}
	return
}

// Report represents a user reporting another user or snippet
type Report struct {
	ID         string    `gorm:"primaryKey;type:text;default:uuid_generate_v4()" json:"id"`
	ReporterID string    `json:"reporterId"`
	Reporter   User      `gorm:"foreignKey:ReporterID" json:"reporter"`
	TargetID   string    `json:"targetId"`   // User ID or Snippet ID
	TargetType string    `json:"targetType"` // "USER" or "SNIPPET"
	Reason     string    `json:"reason"`
	Status     string    `gorm:"default:'PENDING'" json:"status"` // PENDING, RESOLVED, DISMISSED
	CreatedAt  time.Time `json:"createdAt"`
}

func (Report) TableName() string {
	return "Report"
}

func (r *Report) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}
