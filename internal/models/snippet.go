package models

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Snippet struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deletedAt" json:"-"`

	Title             string         `gorm:"uniqueIndex" json:"title"`
	Description       string         `json:"description"`
	Language          string         `json:"language"`
	Code              string         `gorm:"type:text" json:"code"`
	Usage             string         `gorm:"type:text" json:"usage"`
	Tags              pq.StringArray `gorm:"type:text[]" json:"tags"`            // Array of strings
	Visibility        string         `gorm:"default:'public'" json:"visibility"` // public/private
	AllowForks        bool           `gorm:"default:true" json:"allowForks"`
	Output            string         `gorm:"type:text" json:"output"` // Deprecated: Use OutputSnapshot
	OutputSnapshot    string         `gorm:"type:text" json:"outputSnapshot"`
	PreviewType       string         `gorm:"default:'TERMINAL'" json:"previewType"` // TERMINAL, WEB_PREVIEW
	ExecutionLanguage string         `gorm:"column:executionLanguage" json:"executionLanguage"`
	Runtime           float64        `gorm:"default:0" json:"runtime"` // ms

	// MVP v1.1: Rich Attributes
	Type       string `gorm:"default:'ALGORITHM'" json:"type"`    // ALGORITHM, UTILITY, EXAMPLE, VISUAL
	Difficulty string `gorm:"default:'MEDIUM'" json:"difficulty"` // EASY, MEDIUM, HARD

	// MVP v1.1: Stats & Signals
	ViewsCount int  `gorm:"default:0" json:"viewsCount"`
	ForkCount  int  `gorm:"default:0" json:"forkCount"`
	CopyCount  int  `gorm:"default:0" json:"copyCount"` // v1.2: Track clipboard copies
	IsFeatured bool `gorm:"default:false" json:"isFeatured"`

	// Execution Validation
	Status              string `gorm:"default:'DRAFT'" json:"status"` // DRAFT, PUBLISHED
	Verified            bool   `gorm:"default:false" json:"verified"`
	LastExecutionStatus string `gorm:"column:lastExecutionStatus" json:"lastExecutionStatus"` // SUCCESS, FAILURE
	LastExecutionOutput string `gorm:"type:text;column:lastExecutionOutput" json:"lastExecutionOutput"`

	// Relations
	AuthorID string `gorm:"column:authorId" json:"authorId"`
	Author   User   `gorm:"foreignKey:AuthorID" json:"author"`

	ForkedFromID *string  `gorm:"column:forkedFromId" json:"forkedFromId"`
	ForkedFrom   *Snippet `gorm:"foreignKey:ForkedFromID" json:"forkedFrom,omitempty"`
}

func (Snippet) TableName() string {
	return "Snippet"
}

// BeforeSave enforces strict constraints on the model
func (s *Snippet) BeforeSave(tx *gorm.DB) (err error) {
	// Constraints for Publishing
	if s.Status == "PUBLISHED" {
		if !s.Verified {
			return errors.New("violation: cannot publish unverified snippet")
		}
		if s.LastExecutionStatus != "SUCCESS" {
			return errors.New("violation: cannot publish failed execution")
		}
	}
	return nil
}
