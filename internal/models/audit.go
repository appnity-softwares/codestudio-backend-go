package models

import "time"

type ActionType string

const (
	ActionStartContest   ActionType = "START_CONTEST"
	ActionFreezeContest  ActionType = "FREEZE_CONTEST"
	ActionEndContest     ActionType = "END_CONTEST"
	ActionWarnSubmission ActionType = "WARN_SUBMISSION"
	ActionDisqualifySub  ActionType = "DISQUALIFY_SUBMISSION"
	ActionBanUser        ActionType = "BAN_USER"
	ActionIgnoreFlag     ActionType = "IGNORE_FLAG"
	ActionWarnUser       ActionType = "WARN_USER"
	// v1.2: New admin actions
	ActionPinSnippet    ActionType = "PIN_SNIPPET"
	ActionAdjustTrust   ActionType = "ADJUST_TRUST"
	ActionCreateContest ActionType = "CREATE_CONTEST"
	ActionUpdateContest ActionType = "UPDATE_CONTEST"
	ActionDeleteContest ActionType = "DELETE_CONTEST"

	ActionCreateProblem   ActionType = "CREATE_PROBLEM"
	ActionUpdateProblem   ActionType = "UPDATE_PROBLEM"
	ActionDeleteProblem   ActionType = "DELETE_PROBLEM"
	ActionReorderProblems ActionType = "REORDER_PROBLEMS"

	ActionUpdateUser        ActionType = "UPDATE_USER"
	ActionDeleteUser        ActionType = "DELETE_USER"
	ActionUpdatePermissions ActionType = "UPDATE_PERMISSIONS"
	ActionDeleteSnippet     ActionType = "DELETE_SNIPPET"
	ActionManageSystem      ActionType = "MANAGE_SYSTEM"
	ActionManageModeration  ActionType = "MANAGE_MODERATION"
)

type AdminAction struct {
	ID         string     `gorm:"primaryKey;type:text" json:"id"`
	AdminID    string     `json:"adminId"`
	Action     ActionType `json:"action"`
	TargetID   string     `json:"targetId"`   // Potentially polymorphic or just an ID string
	TargetType string     `json:"targetType"` // "contest", "submission", "user"
	Reason     string     `json:"reason"`
	Details    string     `json:"details"` // JSON string
	CreatedAt  time.Time  `json:"createdAt"`

	Admin User `gorm:"foreignKey:AdminID" json:"admin"`
}
