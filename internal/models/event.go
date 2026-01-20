package models

import (
	"time"
)

type EventStatus string

const (
	EventStatusDraft    EventStatus = "DRAFT"
	EventStatusUpcoming EventStatus = "UPCOMING"
	EventStatusLive     EventStatus = "LIVE"
	EventStatusFrozen   EventStatus = "FROZEN"
	EventStatusEnded    EventStatus = "ENDED"
)

type Event struct {
	ID          string `gorm:"primaryKey;type:text" json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Slug        string `gorm:"uniqueIndex" json:"slug"` // URL friendly
	Banner      string `json:"banner"`

	Type        string `gorm:"default:'INTERNAL'" json:"type"` // INTERNAL, EXTERNAL
	ExternalURL string `json:"externalUrl"`

	StartTime  time.Time  `json:"startTime"`
	EndTime    time.Time  `json:"endTime"`
	FreezeTime *time.Time `json:"freezeTime"` // Optional, nil if no freeze

	Price  float64     `json:"price"` // 0 for free
	Status EventStatus `gorm:"type:text;default:'UPCOMING'" json:"status"`

	// --- External Contest Fields ---
	IsExternal            bool      `gorm:"column:isExternal;default:false" json:"isExternal"`
	ExternalPlatform      string    `gorm:"column:externalPlatform" json:"externalPlatform"` // HACKERRANK, CODEFORCES, CUSTOM
	ExternalJoinURL       string    `gorm:"column:externalJoinUrl" json:"externalJoinUrl"`   // Masked in standard API
	ExternalJoinVisibleAt time.Time `gorm:"column:externalJoinVisibleAt" json:"externalJoinVisibleAt"`

	CreatedBy string `json:"createdBy"`
	Creator   User   `gorm:"foreignKey:CreatedBy" json:"creator"`

	Problems []Problem `gorm:"foreignKey:EventID" json:"problems,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Problem struct {
	ID          string `gorm:"primaryKey;type:text" json:"id"`
	EventID     string `json:"eventId"`
	Title       string `json:"title"`
	Description string `json:"description"` // Markdown support
	Difficulty  string `json:"difficulty"`  // Easy, Medium, Hard
	Points      int    `json:"points"`

	// Constraints
	TimeLimit   float64 `json:"timeLimit"`   // in seconds
	MemoryLimit int     `json:"memoryLimit"` // in MB
	Penalty     int     `json:"penalty"`     // in minutes (default 10)

	// Content
	StarterCode string     `json:"starterCode"` // JSON map[lang]code or just string
	TestCases   []TestCase `gorm:"foreignKey:ProblemID" json:"testCases,omitempty"`

	Order int `json:"order"`
}

type TestCase struct {
	ID        string `gorm:"primaryKey;type:text" json:"id"`
	ProblemID string `json:"problemId"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	IsHidden  bool   `json:"isHidden"` // Public vs Private test cases
}

type RegistrationStatus string

const (
	RegStatusPending RegistrationStatus = "PENDING"
	RegStatusPaid    RegistrationStatus = "PAID"
	RegStatusJoined  RegistrationStatus = "JOINED"
	RegStatusNoShow  RegistrationStatus = "NO_SHOW"
)

type Registration struct {
	ID      string `gorm:"primaryKey;type:text" json:"id"`
	UserID  string `gorm:"uniqueIndex:idx_event_user" json:"userId"`
	EventID string `gorm:"uniqueIndex:idx_event_user" json:"eventId"`

	Status    RegistrationStatus `gorm:"type:text" json:"status"`
	PaymentID string             `json:"paymentId"` // Razorpay Payment ID or Order ID reference

	RulesAccepted   bool      `json:"rulesAccepted"`
	RulesAcceptedAt time.Time `json:"rulesAcceptedAt"`

	Score int `json:"score"`
	Rank  int `json:"rank"`

	JoinedExternalAt *time.Time `gorm:"column:joinedExternalAt" json:"joinedExternalAt"`

	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Event Event `gorm:"foreignKey:EventID" json:"event,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

type SubmissionStatus string

const (
	SubStatusAC      SubmissionStatus = "ACCEPTED"
	SubStatusWA      SubmissionStatus = "WRONG_ANSWER"
	SubStatusTLE     SubmissionStatus = "TIME_LIMIT_EXCEEDED"
	SubStatusRE      SubmissionStatus = "RUNTIME_ERROR"
	SubStatusCE      SubmissionStatus = "COMPILATION_ERROR"
	SubStatusPending SubmissionStatus = "PENDING"
)

type Submission struct {
	ID        string `gorm:"primaryKey;type:text" json:"id"`
	UserID    string `json:"userId"`
	EventID   string `json:"eventId"`
	ProblemID string `json:"problemId"`

	Code     string `json:"code"`
	Language string `json:"language"`

	Status  SubmissionStatus `gorm:"type:text" json:"status"`
	Verdict string           `json:"verdict"` // Detailed message

	Runtime float64 `json:"runtime"` // ms
	Memory  int     `json:"memory"`  // KB/MB

	TestCasesPassed int `json:"testCasesPassed"`
	TotalTestCases  int `json:"totalTestCases"`

	OutputSnapshot string `gorm:"type:text" json:"outputSnapshot"` // Full execution result

	// Anti-Cheat
	CodeHash string `gorm:"type:text;index" json:"-"` // SHA-256 hash for similarity detection

	CreatedAt time.Time `json:"createdAt"`

	User    User              `gorm:"foreignKey:UserID" json:"-"`
	Problem Problem           `gorm:"foreignKey:ProblemID" json:"-"`
	Flags   []SubmissionFlag  `gorm:"foreignKey:SubmissionID" json:"flags,omitempty"`
	Metrics SubmissionMetrics `gorm:"foreignKey:SubmissionID" json:"metrics,omitempty"`
}

type SubmissionFlagType string

const (
	FlagTypePaste      SubmissionFlagType = "PASTE_ATTEMPT"
	FlagTypeBlur       SubmissionFlagType = "FOCUS_LOST"
	FlagTypeHash       SubmissionFlagType = "DUPLICATE_HASH"
	FlagTypeSuspicious SubmissionFlagType = "SUSPICIOUS_PATTERN"
)

type SubmissionFlag struct {
	ID           string             `gorm:"primaryKey;type:text" json:"id"`
	SubmissionID string             `json:"submissionId"`
	Type         SubmissionFlagType `json:"type"`
	Details      string             `gorm:"type:text" json:"details"` // JSON or text details
	CreatedAt    time.Time          `json:"createdAt"`
}

type SubmissionMetrics struct {
	SubmissionID string `gorm:"primaryKey;type:text" json:"submissionId"`

	// Behavioral
	PasteCount   int `json:"pasteCount"`
	PastedChars  int `json:"pastedChars"`
	BlurCount    int `json:"blurCount"`
	TabSwitchCnt int `json:"tabSwitchCnt"`

	// Network
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`

	// Structural (Calculated server-side)
	LineCount     int `json:"lineCount"`
	FunctionCount int `json:"functionCount"`
	LoopCount     int `json:"loopCount"`
}
