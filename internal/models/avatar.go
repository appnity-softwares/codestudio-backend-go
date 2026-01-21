package models

import (
	"time"
)

type AvatarSeed struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Seed      string    `gorm:"unique;not null" json:"seed"`
	Style     string    `gorm:"default:'avataaars'" json:"style"` // e.g., avataaars, bottts, etc.
	CreatedAt time.Time `json:"createdAt"`
	AddedBy   string    `json:"addedBy"`
}
