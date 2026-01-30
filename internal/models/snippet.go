package models

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Snippet struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time      `gorm:"column:createdAt;index" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deletedAt" json:"-"`

	Title             string         `gorm:"uniqueIndex" json:"title"`
	Description       string         `json:"description"`
	Language          string         `json:"language"`
	Code              string         `gorm:"type:text" json:"code"`
	Usage             string         `gorm:"type:text" json:"usage"`
	Tags              pq.StringArray `gorm:"type:text[];index" json:"tags"`            // Array of strings
	Visibility        string         `gorm:"default:'public';index" json:"visibility"` // public/private
	Output            string         `gorm:"type:text" json:"output"`                  // Deprecated: Use OutputSnapshot
	OutputSnapshot    string         `gorm:"type:text" json:"outputSnapshot"`
	PreviewType       string         `gorm:"default:'TERMINAL'" json:"previewType"` // TERMINAL, WEB_PREVIEW
	ReferenceURL      string         `json:"referenceUrl"`
	ExecutionLanguage string         `gorm:"column:executionLanguage" json:"executionLanguage"`
	Runtime           float64        `gorm:"default:0" json:"runtime"` // ms

	// MVP v1.1: Rich Attributes
	Type       string `gorm:"default:'ALGORITHM';index" json:"type"`    // ALGORITHM, UTILITY, EXAMPLE, VISUAL
	Difficulty string `gorm:"default:'MEDIUM';index" json:"difficulty"` // EASY, MEDIUM, HARD

	// MVP v1.1: Stats & Signals
	ViewsCount    int  `gorm:"default:0" json:"viewsCount"`
	CopyCount     int  `gorm:"default:0" json:"copyCount"`
	LikesCount    int  `gorm:"default:0" json:"likesCount"`    // v1.3 Engagement
	DislikesCount int  `gorm:"default:0" json:"dislikesCount"` // v1.4
	IsFeatured    bool `gorm:"default:false;index" json:"isFeatured"`

	// Virtual Fields (Auth Context)
	ViewerReaction string `gorm:"-" json:"viewerReaction"` // "like", "dislike", or ""

	// Execution Validation
	Status              string `gorm:"default:'DRAFT'" json:"status"` // DRAFT, PUBLISHED
	Verified            bool   `gorm:"default:false" json:"verified"`
	LastExecutionStatus string `gorm:"column:lastExecutionStatus" json:"lastExecutionStatus"` // SUCCESS, FAILURE
	LastExecutionOutput string `gorm:"type:text;column:lastExecutionOutput" json:"lastExecutionOutput"`
	Annotations         string `gorm:"type:text" json:"annotations"`  // JSON string of line annotations
	StdinHistory        string `gorm:"type:text" json:"stdinHistory"` // JSON string of interactive session

	// Relations
	AuthorID     string   `gorm:"column:authorId;index" json:"authorId"`
	Author       User     `gorm:"foreignKey:AuthorID" json:"author"`
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
