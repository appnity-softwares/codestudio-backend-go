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

// RolePermission defines permissions for a specific role
type RolePermission struct {
	Role              Role      `gorm:"primaryKey;type:text" json:"role"`
	CanManageUsers    bool      `gorm:"default:false" json:"canManageUsers"`
	CanManageSnippets bool      `gorm:"default:false" json:"canManageSnippets"`
	CanManageContests bool      `gorm:"default:false" json:"canManageContests"`
	CanManageProblems bool      `gorm:"default:false" json:"canManageProblems"`
	CanViewAuditLogs  bool      `gorm:"default:false" json:"canViewAuditLogs"`
	CanManageSystem   bool      `gorm:"default:false" json:"canManageSystem"` // Usually Admin only
	UpdatedAt         time.Time `json:"updatedAt"`
	UpdatedBy         string    `json:"updatedBy"`
}

// System setting keys (constants for type safety)
const (
	SettingMaintenanceMode    = "maintenance_mode"
	SettingSubmissionsEnabled = "submissions_enabled"
	SettingSnippetsEnabled    = "snippets_enabled"
	SettingContestsEnabled    = "contests_enabled"
	SettingRegistrationOpen   = "registration_open"
	SettingMaintenanceETA     = "maintenance_eta"

	// Feature Flags (v1.3)
	SettingFeatureSidebarXPStore       = "feature_sidebar_xp_store"
	SettingFeatureSidebarTrophyRoom    = "feature_sidebar_trophy_room"
	SettingFeatureSidebarPractice      = "feature_sidebar_practice"
	SettingFeatureSidebarFeedback      = "feature_sidebar_feedback"
	SettingFeatureSidebarRoadmaps      = "feature_sidebar_roadmaps"
	SettingFeatureSidebarCommunity     = "feature_sidebar_community"
	SettingFeatureInterfaceEngine      = "feature_interface_engine"
	SettingFeatureQuestsEnabled        = "feature_quests_enabled"
	SettingFeatureSidebarLeaderboard   = "feature_sidebar_leaderboard"
	SettingFeatureNotificationsEnabled = "feature_notifications_enabled"
	SettingFeatureSidebarNewBadge      = "feature_sidebar_new_badge"
	SettingSidebarBadges               = "sidebar_badges"
	SettingFeatureGithubStats          = "feature_github_stats"
	SettingFeatureSocialChat           = "feature_social_chat"
	SettingFeatureSocialFollow         = "feature_social_follow"
	SettingFeatureSocialFeed           = "feature_social_feed"

	// System Banner
	SettingBannerVisible = "system_banner_visible"
	SettingBannerTitle   = "system_banner_title"
	SettingBannerBadge   = "system_banner_badge"
	SettingBannerContent = "system_banner_content" // JSON or Text
	SettingBannerLink    = "system_banner_link"
)

// AdminAuditLog extends AdminAction with IP tracking (used for detailed audit)
type AdminAuditLog struct {
	ID         string     `gorm:"primaryKey;type:text" json:"id"`
	AdminID    string     `gorm:"index" json:"adminId"`
	ActionType ActionType `json:"actionType"`
	EntityType string     `json:"entityType"` // "user", "contest", "submission", "snippet", "system"
	EntityID   string     `json:"entityId"`
	Metadata   *string    `gorm:"type:jsonb" json:"metadata"` // JSON blob for extra details
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

	// v1.3 Enhanced Metrics
	NewSnippetsToday      int64   `json:"newSnippetsToday"`
	SubmissionSuccessRate float64 `json:"submissionSuccessRate"`
}
