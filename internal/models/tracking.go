package models

import "time"

// EntityType for tracking different content types
type EntityType string

const (
	EntityTypeSnippet EntityType = "SNIPPET"
	EntityTypeProblem EntityType = "PROBLEM"
	EntityTypeContest EntityType = "CONTEST"
)

// EntityView tracks unique views per user per entity
// One user can only view an entity once for counting purposes
type EntityView struct {
	ID         string     `gorm:"primaryKey;type:text" json:"id"`
	UserID     string     `gorm:"index;type:text;not null" json:"userId"`
	EntityType EntityType `gorm:"type:text;not null" json:"entityType"`
	EntityID   string     `gorm:"type:text;not null" json:"entityId"`
	ViewedAt   time.Time  `gorm:"autoCreateTime" json:"viewedAt"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName sets the table name for GORM
func (EntityView) TableName() string {
	return "entity_views"
}

// EntityCopy tracks unique copy actions per user per entity
// One user can only copy an entity once for counting purposes
type EntityCopy struct {
	ID         string     `gorm:"primaryKey;type:text" json:"id"`
	UserID     string     `gorm:"index;type:text;not null" json:"userId"`
	EntityType EntityType `gorm:"type:text;not null" json:"entityType"`
	EntityID   string     `gorm:"type:text;not null" json:"entityId"`
	CopiedAt   time.Time  `gorm:"autoCreateTime" json:"copiedAt"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName sets the table name for GORM
func (EntityCopy) TableName() string {
	return "entity_copies"
}
