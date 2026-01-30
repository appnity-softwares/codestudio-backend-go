package models

import (
	"time"
)

type ShortLink struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Code        string     `gorm:"uniqueIndex;not null" json:"code"`
	OriginalURL string     `gorm:"not null" json:"originalUrl"`
	Visits      int        `gorm:"default:0" json:"visits"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}
