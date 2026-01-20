package models

import "time"

type BadgeCategory string

const (
	BadgeCategorySystem BadgeCategory = "SYSTEM"
	BadgeCategorySkill  BadgeCategory = "SKILL"
	BadgeCategoryTrust  BadgeCategory = "TRUST"
)

type Badge struct {
	ID          string        `gorm:"primaryKey;type:text" json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"` // Name of the Lucide icon
	Category    BadgeCategory `gorm:"type:text" json:"category"`
	Condition   string        `json:"condition"` // e.g. "5_snippets"
	Threshold   int           `json:"threshold"`
}

type UserBadge struct {
	UserID     string    `gorm:"primaryKey;type:text" json:"userId"`
	BadgeID    string    `gorm:"primaryKey;type:text" json:"badgeId"`
	Progress   int       `gorm:"default:0" json:"progress"`
	UnlockedAt time.Time `json:"unlockedAt"`

	Badge Badge `gorm:"foreignKey:BadgeID" json:"badge"`
	User  User  `gorm:"foreignKey:UserID" json:"-"`
}
