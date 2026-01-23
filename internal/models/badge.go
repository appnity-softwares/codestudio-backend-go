package models

import "time"

type BadgeCategory string
type BadgeType string

const (
	BadgeCategorySystem BadgeCategory = "SYSTEM"
	BadgeCategorySkill  BadgeCategory = "SKILL"
	BadgeCategoryTrust  BadgeCategory = "TRUST"

	BadgeType2D BadgeType = "BADGE"
	BadgeType3D BadgeType = "TROPHY"
)

type Badge struct {
	ID          string        `gorm:"primaryKey;type:text" json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"` // Name of the Lucide icon
	Category    BadgeCategory `gorm:"type:text" json:"category"`
	Type        BadgeType     `gorm:"type:text;default:'BADGE'" json:"type"`
	Condition   string        `json:"condition"` // e.g. "5_snippets"
	Threshold   int           `json:"threshold"`
	ModelPath   string        `json:"modelPath"` // Path to 3D model if Type is TROPHY
}

type UserBadge struct {
	UserID     string    `gorm:"primaryKey;type:text" json:"userId"`
	BadgeID    string    `gorm:"primaryKey;type:text" json:"badgeId"`
	Progress   int       `gorm:"default:0" json:"progress"`
	UnlockedAt time.Time `json:"unlockedAt"`

	Badge Badge `gorm:"foreignKey:BadgeID" json:"badge"`
	User  User  `gorm:"foreignKey:UserID" json:"-"`
}
