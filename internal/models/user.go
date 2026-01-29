package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Role string

const (
	RoleUser      Role = "USER"
	RoleAdmin     Role = "ADMIN"
	RoleModerator Role = "MODERATOR"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "PUBLIC"
	VisibilityPrivate Visibility = "PRIVATE"
	VisibilityHybrid  Visibility = "HYBRID"
)

const XPPerLevel = 1000 // 1000 XP per level

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
	LinkedInURL   string     `gorm:"column:linkedinUrl" json:"linkedinUrl"`
	IsBlocked     bool       `gorm:"default:false" json:"isBlocked"`
	VaultKey      string     `gorm:"column:vaultKey" json:"vaultKey"`

	// Enums stored as strings
	Role       Role       `gorm:"type:text;default:'USER'" json:"role"`
	Visibility Visibility `gorm:"type:text;default:'PUBLIC'" json:"visibility"`

	OnboardingCompleted bool           `gorm:"default:false" json:"onboardingCompleted"`
	PreferredLanguages  pq.StringArray `gorm:"type:text[]" json:"preferredLanguages"` // Postgres Array
	Interests           pq.StringArray `gorm:"type:text[]" json:"interests"`          // Postgres Array
	IsModerator         bool           `gorm:"default:false" json:"isModerator"`

	// Anti-Cheat (MVP)
	TrustScore int `gorm:"default:100" json:"trustScore"`

	// Arrays (Postgres String Array)
	SelectedPublicSnippetIds pq.StringArray `gorm:"type:text[]" json:"selectedPublicSnippetIds"` // GORM might need custom handling for arrays or simpler approach
	PurchasedComponentIds    pq.StringArray `gorm:"type:text[]" json:"purchasedComponentIds"`

	// MVP v1.1: Profile Customization
	PinnedSnippetID *string  `gorm:"column:pinnedSnippetId" json:"pinnedSnippetId"`
	PinnedSnippet   *Snippet `gorm:"foreignKey:PinnedSnippetID" json:"pinnedSnippet,omitempty"`

	// v1.3: Engagement & Social
	City         string         `json:"city"`
	Endorsements pq.StringArray `gorm:"type:text[]" json:"endorsements"`

	// Identity Management
	UsernameChangeCount  int        `gorm:"default:0" json:"usernameChangeCount"`
	LastUsernameChangeAt *time.Time `json:"lastUsernameChangeAt"`

	// Privacy settings
	PublicProfileEnabled bool `gorm:"default:true" json:"publicProfileEnabled"`
	SearchVisible        bool `gorm:"default:true" json:"searchVisible"`
	GithubStatsVisible   bool `gorm:"default:true;column:githubStatsVisible" json:"githubStatsVisible"`

	// Cached Counters (for Leaderboard/Community performance)
	WrappedSnippetCount int    `gorm:"default:0;column:snippet_count" json:"snippetCount"`
	WrappedViewCount    int    `gorm:"default:0;column:view_count" json:"viewCount"`
	WrappedContestCount int    `gorm:"default:0;column:contest_count" json:"contestCount"`
	XP                  int    `gorm:"default:0" json:"xp"`
	Level               int    `gorm:"default:1" json:"level"`
	EquippedAura        string `gorm:"column:equippedAura" json:"equippedAura"`

	// Social & Engagement (Cached)
	LinkersCount int `gorm:"default:0;column:linkersCount" json:"linkersCount"` // Followers
	LinkedCount  int `gorm:"default:0;column:linkedCount" json:"linkedCount"`   // Following

	// Future Integrations
	GithubStats *string `gorm:"type:jsonb" json:"githubStats"` // JSONB for flexible stats

	ResetToken       string     `json:"-"`
	ResetTokenExpiry *time.Time `json:"-"`

	Password string `json:"-"`

	Count       UserCount `gorm:"-" json:"_count"`
	IsFollowing bool      `gorm:"-" json:"isFollowing"`
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

// SyncLevelXP ensures XP and Level are consistent.
// If mode is "XP", Level is updated based on XP.
// If mode is "Level", XP is updated to the minimum XP for that level if current XP is lower.
func (u *User) SyncLevelXP(mode string) {
	switch mode {
	case "XP":
		u.Level = (u.XP / XPPerLevel) + 1
	case "Level":
		minXP := (u.Level - 1) * XPPerLevel
		if u.XP < minXP {
			u.XP = minXP
		}
	}
}
