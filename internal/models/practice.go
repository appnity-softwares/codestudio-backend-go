package models

import (
	"time"

	"gorm.io/gorm"
)

// PracticeProblem represents a non-official practice problem
// These are separate from contest problems and have no time limits or anti-cheat
type PracticeProblem struct {
	ID        string         `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deletedAt" json:"-"`

	Title       string `json:"title"`
	Description string `gorm:"type:text" json:"description"`
	Difficulty  string `gorm:"default:'MEDIUM'" json:"difficulty"` // EASY, MEDIUM, HARD
	Category    string `json:"category"`                           // Arrays, Strings, Trees, etc.

	// Problem content
	StarterCode  string `gorm:"type:text" json:"starterCode"`
	SolutionCode string `gorm:"type:text" json:"solutionCode"`  // Hidden from users
	TestCases    string `gorm:"type:text" json:"testCases"`     // JSON array of test cases
	Language     string `json:"language"`                       // Default language
	TimeLimit    int    `gorm:"default:2" json:"timeLimit"`     // Seconds
	MemoryLimit  int    `gorm:"default:128" json:"memoryLimit"` // MB

	// Metadata
	IsDailyProblem bool `gorm:"default:false" json:"isDailyProblem"`
	SolveCount     int  `gorm:"default:0" json:"solveCount"`
	AttemptCount   int  `gorm:"default:0" json:"attemptCount"`

	// Relations
	CreatorID string `gorm:"column:creatorId" json:"creatorId"`
	Creator   User   `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
}

func (PracticeProblem) TableName() string {
	return "practice_problems"
}

// PracticeSubmission represents a user's attempt at a practice problem
type PracticeSubmission struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`

	// Relations
	UserID    string          `gorm:"column:userId" json:"userId"`
	User      User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ProblemID string          `gorm:"column:problemId" json:"problemId"`
	Problem   PracticeProblem `gorm:"foreignKey:ProblemID" json:"problem,omitempty"`

	// Submission data
	Code     string `gorm:"type:text" json:"code"`
	Language string `json:"language"`

	// Result
	Status        string `gorm:"default:'PENDING'" json:"status"` // PENDING, RUNNING, ACCEPTED, WRONG_ANSWER, ERROR, TIMEOUT
	Verdict       string `json:"verdict"`
	ExecutionTime int    `json:"executionTime"` // ms
	MemoryUsed    int    `json:"memoryUsed"`    // KB
	Output        string `gorm:"type:text" json:"output"`
	Error         string `gorm:"type:text" json:"error"`

	// Test results
	TestsPassed int `json:"testsPassed"`
	TestsTotal  int `json:"testsTotal"`
}

func (PracticeSubmission) TableName() string {
	return "practice_submissions"
}
