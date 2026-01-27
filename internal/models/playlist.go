package models

import (
	"time"

	"gorm.io/gorm"
)

type Playlist struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deletedAt" json:"-"`

	Title       string `gorm:"uniqueIndex" json:"title"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	Difficulty  string `gorm:"default:'BEGINNER'" json:"difficulty"` // BEGINNER, INTERMEDIATE, ADVANCED

	AuthorID string `gorm:"column:authorId" json:"authorId"`
	Author   User   `gorm:"foreignKey:AuthorID" json:"author"`

	Items []PlaylistSnippet `gorm:"foreignKey:PlaylistID" json:"items"`

	IsPublished bool `gorm:"default:false" json:"isPublished"`
	ViewsCount  int  `gorm:"default:0" json:"viewsCount"`

	// v1.3: Certifications
	// v1.3: Certifications
	IsVerified        bool   `gorm:"default:false" json:"isVerified"`
	AwardsEndorsement string `json:"awardsEndorsement"`
	CompletionBonusXP int    `gorm:"default:0" json:"completionBonusXP"`
}

type PlaylistSnippet struct {
	ID          string  `gorm:"primaryKey;type:text" json:"id"`
	PlaylistID  string  `gorm:"index" json:"playlistId"`
	SnippetID   string  `json:"snippetId"`
	Snippet     Snippet `gorm:"foreignKey:SnippetID" json:"snippet"`
	Order       int     `json:"order"`
	IsCompleted bool    `gorm:"-" json:"isCompleted"`
}

func (Playlist) TableName() string {
	return "Playlist"
}

func (PlaylistSnippet) TableName() string {
	return "PlaylistSnippet"
}
