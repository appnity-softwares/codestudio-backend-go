package models

import (
	"time"

	"github.com/lib/pq"
)

// ChangelogEntry represents a version release note
type ChangelogEntry struct {
	ID          string     `gorm:"primaryKey;type:text" json:"id"`
	Version     string     `gorm:"index" json:"version"`
	Title       string     `json:"title"`
	Description string     `gorm:"type:text" json:"description"` // Markdown content
	ReleaseType string     `json:"releaseType"`                  // FEATURE, FIX, SECURITY, BREAKING
	IsPublished bool       `gorm:"default:false;index" json:"isPublished"`
	ReleasedAt  *time.Time `gorm:"index" json:"releasedAt"`
	Order       int        `gorm:"default:0" json:"order"`
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy"` // Admin ID
	// Legacy support (optional, can remove if we migrate data)
	Changes pq.StringArray `gorm:"type:text[]" json:"changes,omitempty"`
}
