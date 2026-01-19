package models

import "time"

// SuspensionType defines whether a suspension is temporary or permanent
type SuspensionType string

const (
	SuspensionTemporary SuspensionType = "TEMPORARY"
	SuspensionPermanent SuspensionType = "PERMANENT"
)

// UserSuspension tracks user account suspensions
type UserSuspension struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	UserID    string         `gorm:"index" json:"userId"`
	AdminID   string         `json:"adminId"`
	Type      SuspensionType `gorm:"type:text" json:"type"`
	Reason    string         `json:"reason"`
	ExpiresAt *time.Time     `json:"expiresAt"` // nil for permanent
	CreatedAt time.Time      `json:"createdAt"`
	LiftedAt  *time.Time     `json:"liftedAt"` // nil if still active
	LiftedBy  *string        `json:"liftedBy"` // Admin who lifted

	User  User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Admin User `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

// TrustScoreHistory tracks all trust score changes for auditing
type TrustScoreHistory struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	UserID    string    `gorm:"index" json:"userId"`
	AdminID   *string   `json:"adminId"` // nil for automated changes
	OldScore  int       `json:"oldScore"`
	NewScore  int       `json:"newScore"`
	Reason    string    `json:"reason"`
	Source    string    `json:"source"` // "ADMIN", "SYSTEM", "CONTEST"
	CreatedAt time.Time `json:"createdAt"`

	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Admin *User `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

// SystemSettings stores global configuration toggles
type SystemSettings struct {
	Key       string    `gorm:"primaryKey;type:text" json:"key"`
	Value     string    `json:"value"`
	UpdatedBy string    `json:"updatedBy"`
	UpdatedAt time.Time `json:"updatedAt"`

	Admin User `gorm:"foreignKey:UpdatedBy" json:"admin,omitempty"`
}

// System setting keys (constants for type safety)
const (
	SettingMaintenanceMode    = "maintenance_mode"
	SettingSubmissionsEnabled = "submissions_enabled"
	SettingSnippetsEnabled    = "snippets_enabled"
	SettingContestsEnabled    = "contests_enabled"
	SettingRegistrationOpen   = "registration_open"
)

// AdminAuditLog extends AdminAction with IP tracking (used for detailed audit)
type AdminAuditLog struct {
	ID         string     `gorm:"primaryKey;type:text" json:"id"`
	AdminID    string     `gorm:"index" json:"adminId"`
	ActionType ActionType `json:"actionType"`
	EntityType string     `json:"entityType"` // "user", "contest", "submission", "snippet", "system"
	EntityID   string     `json:"entityId"`
	Metadata   string     `gorm:"type:jsonb" json:"metadata"` // JSON blob for extra details
	IPAddress  string     `json:"ipAddress"`
	UserAgent  string     `json:"userAgent"`
	CreatedAt  time.Time  `gorm:"index" json:"createdAt"`

	Admin User `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

// DashboardMetrics is a helper struct for dashboard API (not persisted)
type DashboardMetrics struct {
	TotalUsers         int64 `json:"totalUsers"`
	ActiveUsersToday   int64 `json:"activeUsersToday"`
	TotalSnippets      int64 `json:"totalSnippets"`
	TotalContests      int64 `json:"totalContests"`
	LiveContests       int64 `json:"liveContests"`
	FlaggedSubmissions int64 `json:"flaggedSubmissions"`
	LowTrustUsers      int64 `json:"lowTrustUsers"`
	TotalSubmissions   int64 `json:"totalSubmissions"`
	PendingSubmissions int64 `json:"pendingSubmissions"`
	SuspendedUsers     int64 `json:"suspendedUsers"`
}
