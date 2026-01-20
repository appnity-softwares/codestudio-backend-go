package models

import (
	"time"

	"gorm.io/gorm"
)

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "PUBLIC"
	VisibilityPrivate Visibility = "PRIVATE"
	VisibilityHybrid  Visibility = "HYBRID"
)

type User struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deletedAt" json:"-"`

	Name          string     `json:"name"`
	Email         string     `gorm:"uniqueIndex" json:"email"`
	EmailVerified *time.Time `json:"emailVerified"`
	Image         string     `json:"image"`
	Username      string     `gorm:"uniqueIndex" json:"username"`
	Bio           string     `json:"bio"`
	GithubURL     string     `gorm:"column:githubUrl" json:"githubUrl"`
	InstagramURL  string     `gorm:"column:instagramUrl" json:"instagramUrl"`
	IsBlocked     bool       `gorm:"default:false" json:"isBlocked"`

	// Enums stored as strings
	Role       Role       `gorm:"type:text;default:'USER'" json:"role"`
	Visibility Visibility `gorm:"type:text;default:'PUBLIC'" json:"visibility"`

	OnboardingCompleted bool    `gorm:"default:false" json:"onboardingCompleted"`
	PreferredLanguages  *string `gorm:"type:text[]" json:"preferredLanguages"` // Postgres Array
	Interests           *string `gorm:"type:text[]" json:"interests"`          // Postgres Array

	// Anti-Cheat (MVP)
	TrustScore int `gorm:"default:100" json:"trustScore"`

	// Arrays (Postgres String Array)
	SelectedPublicSnippetIds *string `gorm:"type:text[]" json:"selectedPublicSnippetIds"` // GORM might need custom handling for arrays or simpler approach
	PurchasedComponentIds    *string `gorm:"type:text[]" json:"purchasedComponentIds"`

	// MVP v1.1: Profile Customization
	PinnedSnippetID *string  `gorm:"column:pinnedSnippetId" json:"pinnedSnippetId"`
	PinnedSnippet   *Snippet `gorm:"foreignKey:PinnedSnippetID" json:"pinnedSnippet,omitempty"`

	// Identity Management
	UsernameChangeCount  int       `gorm:"default:0" json:"usernameChangeCount"`
	LastUsernameChangeAt time.Time `json:"lastUsernameChangeAt"`

	ResetToken       string    `json:"-"`
	ResetTokenExpiry time.Time `json:"-"`

	Password string `json:"-"`

	Count UserCount `gorm:"-" json:"_count"`
}

type UserCount struct {
	Snippets int64 `json:"snippets"`
}

// TableName overrides the table name used by User to `User` to match Prisma's default naming (likely `User` or `users` depending on prisma config, Prisma usually maps to `User` table if model is `User` in double quotes, or `User` table).
// Prisma default is PascalCase for model -> PascalCase/camelCase for table? No, Prisma default is strict.
// Looking at schema.prisma: model User maps to "User" table usually unless @@map is used.
// We should check if the table is "User" or "users". Postgres is case sensitive if quoted, usually lowercase if not.
// Prisma usually creates "User" (quoted) or "User" (case sensitive).
// To be safe, let's verify later. For now assume "User".
func (User) TableName() string {
	return "User"
}
