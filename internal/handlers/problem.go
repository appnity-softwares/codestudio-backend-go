package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/internal/services"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"gorm.io/gorm"
)

// RunSolutionInput is the request body for running code against sample tests
type RunSolutionInput struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
}

// RunSolution executes code against ALL SAMPLE test cases
func RunSolution(c *gin.Context) {
	problemID := c.Param("problemId")

	var input RunSolutionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var problem models.Problem
	if err := database.DB.Preload("TestCases").First(&problem, "id = ?", problemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	var sampleCases []models.TestCase
	for _, tc := range problem.TestCases {
		if !tc.IsHidden {
			sampleCases = append(sampleCases, tc)
		}
	}

	if len(sampleCases) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "No Sample Tests",
			"results": []interface{}{},
		})
		return
	}

	// Result Structure
	type TestCaseResult struct {
		Input    string `json:"input"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
		Status   string `json:"status"` // PASSED, FAILED, ERROR
		Stderr   string `json:"stderr"`
	}

	var results []TestCaseResult

	for _, tc := range sampleCases {
		// Default limits for Run
		res, err := services.ExecuteCode(input.Language, input.Code, tc.Input, 2.0, 128)

		var result TestCaseResult
		result.Input = tc.Input
		result.Expected = tc.Output

		if err != nil {
			result.Status = "ERROR"
			result.Stderr = err.Error()
		} else if res.Run.Code != 0 {
			result.Status = "ERROR"
			result.Stderr = res.Run.Stderr
			if res.Run.Signal != "" {
				result.Stderr += " (Signal: " + res.Run.Signal + ")"
			}
		} else {
			// Normalize check
			actualNormalized := strings.TrimSpace(strings.ReplaceAll(res.Run.Stdout, "\r\n", "\n"))
			expectedNormalized := strings.TrimSpace(strings.ReplaceAll(tc.Output, "\r\n", "\n"))

			result.Actual = res.Run.Stdout
			result.Stderr = res.Run.Stderr

			if actualNormalized == expectedNormalized {
				result.Status = "PASSED"
			} else {
				result.Status = "FAILED"
			}
		}
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"type":    "run",
		"results": results,
	})
}

// ListProblems returns all problems for an event
// ListProblems returns all problems for an event
func ListProblems(c *gin.Context) {
	eventID := c.Param("eventId")
	userID, _ := c.Get("userId") // Optional auth for context, but required for contest rules

	// Practice Arena is always open
	if eventID == "practice-arena-mvp" {
		var problems []models.Problem
		if err := database.DB.Where("event_id = ?", eventID).Order("\"order\" asc").Find(&problems).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"problems": problems})
		return
	}

	// For Official Contests: STRICT CHECK
	var event models.Event
	if err := database.DB.First(&event, "id = ?", eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// 1. Time Lock
	if time.Now().Before(event.StartTime) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Contest has not started yet"})
		return
	}

	// 2. Rules Acceptance Lock (Require Auth)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Login required"})
		return
	}

	// Check Registration & Rules
	var registration models.Registration
	if err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).First(&registration).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not registered"})
		return
	}

	if !registration.RulesAccepted {
		c.JSON(http.StatusForbidden, gin.H{"error": "Rules not accepted", "rulesRequired": true})
		return
	}

	var problems []models.Problem
	// Hide sensitive fields like test cases? Currently DB struct has them but json ignore?
	// The struct doesn't ignore TestCases list completely, but we rely on omitempty or Preload control.
	// Since we are not Preloading TestCases here, they won't be returned (Go default empty slice).
	if err := database.DB.Where("event_id = ?", eventID).Order("\"order\" asc").Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problems": problems})
}

// GetProblem returns a single problem by ID
func GetProblem(c *gin.Context) {
	problemID := c.Param("problemId")

	var problem models.Problem
	if err := database.DB.Preload("TestCases").First(&problem, "id = ?", problemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Preload test cases for execution context on frontend (if admin or needed)
	// Usually hidden test cases are not sent.
	// MVP: Send all test cases but client should hide 'output' if hidden.
	// Secure: Don't preload TestCases here, or filter them.
	// Let's send basic info.
	// Security: Filter hidden test cases or mask outputs
	// For "Sample input/output" requirement, we return public test cases.
	// We should NOT return hidden test cases at all, or return them without Input/Output.
	// Best practice: Return only PUBLIC test cases.
	var publicTestCases []models.TestCase
	for _, tc := range problem.TestCases {
		if !tc.IsHidden {
			publicTestCases = append(publicTestCases, tc)
		}
	}
	problem.TestCases = publicTestCases

	// Security: Fetch Event and check Access
	var event models.Event
	if err := database.DB.First(&event, "id = ?", problem.EventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Practice Arena is open
	if event.ID != "practice-arena-mvp" {
		userID, _ := c.Get("userId")
		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Login required"})
			return
		}

		// 1. Time Lock
		if time.Now().Before(event.StartTime) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Contest has not started yet"})
			return
		}

		// 2. Rules & Registration
		var registration models.Registration
		if err := database.DB.Where("user_id = ? AND event_id = ?", userID, event.ID).First(&registration).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not registered"})
			return
		}
		if !registration.RulesAccepted {
			c.JSON(http.StatusForbidden, gin.H{"error": "Rules not accepted"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"problem": problem})
}

// CreateProblemInput is the request body for creating a problem
type CreateProblemInput struct {
	Title       string            `json:"title" binding:"required"`
	Description string            `json:"description" binding:"required"`
	Difficulty  string            `json:"difficulty"`
	TimeLimit   float64           `json:"timeLimit"`
	MemoryLimit int               `json:"memoryLimit"`
	Points      int               `json:"points"`
	StarterCode string            `json:"starterCode"` // stringified JSON or text
	TestCases   []models.TestCase `json:"testCases"`
	Order       int               `json:"order"`
}

// CreateProblem creates a new problem for an event (Admin only)
func CreateProblem(c *gin.Context) {
	eventID := c.Param("eventId")

	var input CreateProblemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	problem := models.Problem{
		ID:          utils.GenerateID(),
		EventID:     eventID,
		Title:       input.Title,
		Description: input.Description,
		Difficulty:  input.Difficulty,
		TimeLimit:   input.TimeLimit,
		MemoryLimit: input.MemoryLimit,
		Points:      input.Points,
		StarterCode: input.StarterCode,
		TestCases:   input.TestCases,
		Order:       input.Order,
	}

	// Assign IDs to TestCases if missing
	for i := range problem.TestCases {
		if problem.TestCases[i].ID == "" {
			problem.TestCases[i].ID = utils.GenerateID()
		}
	}

	if err := database.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"problem": problem})
}

// UpdateProblem updates an existing problem (Admin only)
func UpdateProblem(c *gin.Context) {
	problemID := c.Param("problemId")

	var problem models.Problem
	if err := database.DB.First(&problem, "id = ?", problemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	var input CreateProblemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Transaction for atomic update
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Update Problem Fields
		if err := tx.Model(&problem).Updates(map[string]interface{}{
			"title":        input.Title,
			"description":  input.Description,
			"difficulty":   input.Difficulty,
			"time_limit":   input.TimeLimit,
			"memory_limit": input.MemoryLimit,
			"points":       input.Points,
			"starter_code": input.StarterCode,
			"order":        input.Order,
		}).Error; err != nil {
			return err
		}

		// 2. Replace Test Cases
		// First, delete existing
		if err := tx.Delete(&models.TestCase{}, "problem_id = ?", problemID).Error; err != nil {
			return err
		}

		// Rewrite new ones
		for _, tc := range input.TestCases {
			newTC := models.TestCase{
				ID:        utils.GenerateID(),
				ProblemID: problemID,
				Input:     tc.Input,
				Output:    tc.Output,
				IsHidden:  tc.IsHidden,
			}
			if err := tx.Create(&newTC).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update problem: " + err.Error()})
		return
	}

	// Reload problem with test cases to return
	database.DB.Preload("TestCases").First(&problem, "id = ?", problemID)
	c.JSON(http.StatusOK, gin.H{"problem": problem})
}

// DeleteProblem deletes a problem (Admin only)
func DeleteProblem(c *gin.Context) {
	problemID := c.Param("problemId")

	if err := database.DB.Delete(&models.Problem{}, "id = ?", problemID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem deleted"})
}

// SubmitSolutionInput is the request for submitting code
type SubmitSolutionInput struct {
	Code        string `json:"code" binding:"required"`
	Language    string `json:"language" binding:"required"`
	PasteCount  int    `json:"pasteCount"`
	PastedChars int    `json:"pastedChars"`
	BlurCount   int    `json:"blurCount"`
}

// SubmitSolution handles code submission for a problem
func SubmitSolution(c *gin.Context) {
	problemID := c.Param("problemId")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(string)

	var input SubmitSolutionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Fetch Problem
	var problem models.Problem
	if err := database.DB.Preload("TestCases").First(&problem, "id = ?", problemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// 2. Access Control & Contest Rules
	var event models.Event
	if err := database.DB.First(&event, "id = ?", problem.EventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// CRITICAL: Contest Time Lock (Server Time Enforcement)
	if event.ID != "practice-arena-mvp" && time.Now().UTC().After(event.EndTime) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Contest has ended. Submissions are no longer accepted."})
		return
	}

	// Rule: Registration Check (skip for practice)
	if event.ID != "practice-arena-mvp" {
		var registration models.Registration
		if err := database.DB.Where("user_id = ? AND event_id = ?", uid, problem.EventID).First(&registration).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You must be registered for this contest"})
			return
		}
	}

	// LAYER 1: HARD RULES (BLOCKING)
	// Rule: Max 20 submissions per problem
	var subCount int64
	database.DB.Model(&models.Submission{}).Where("user_id = ? AND problem_id = ?", uid, problemID).Count(&subCount)
	if subCount >= 20 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Submission limit exceeded (Max 20)"})
		return
	}

	// 3. Strict Boilerplate & Content Validation
	cleanCode := strings.TrimSpace(input.Code)
	if cleanCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty submission rejected"})
		return
	}

	// Check against boilerplates
	isBoilerplate := false
	switch input.Language {
	case "python":
		if strings.Contains(cleanCode, "def solve():") && strings.Contains(cleanCode, "pass") && len(cleanCode) < 60 {
			isBoilerplate = true
		}
	case "go":
		if strings.Contains(cleanCode, "package main") && strings.Contains(cleanCode, "func main()") && len(cleanCode) < 100 {
			isBoilerplate = true
		}
	case "javascript":
		if strings.Contains(cleanCode, "function solve()") && len(cleanCode) < 50 {
			isBoilerplate = true
		}
	case "cpp":
		if strings.Contains(cleanCode, "int main()") && len(strings.ReplaceAll(cleanCode, " ", "")) < 60 {
			isBoilerplate = true
		}
	case "java":
		if strings.Contains(cleanCode, "public class Main") && strings.Contains(cleanCode, "// Your code here") && len(cleanCode) < 120 {
			isBoilerplate = true
		}
	}

	if isBoilerplate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unmodified boilerplate rejected. Write some code!"})
		return
	}

	// 4. Rate Limiting (10s cooldown)
	var lastSub models.Submission
	if err := database.DB.Where("user_id = ? AND problem_id = ?", uid, problemID).Order("created_at desc").First(&lastSub).Error; err == nil {
		if time.Since(lastSub.CreatedAt) < 10*time.Second {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait 10 seconds between submissions"})
			return
		}
	}

	// LAYER 3: STRUCTURAL ANALYSIS
	lineCount := len(strings.Split(cleanCode, "\n"))

	// Strip comments for better hash comparison
	codeNoComments := cleanCode
	// Remove single-line comments
	codeNoComments = regexp.MustCompile(`//.*`).ReplaceAllString(codeNoComments, "")
	codeNoComments = regexp.MustCompile(`#.*`).ReplaceAllString(codeNoComments, "")
	// Remove multi-line comments (simple)
	codeNoComments = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(codeNoComments, "")

	funcCount := 0
	loopCount := 0
	switch input.Language {
	case "python":
		funcCount = strings.Count(cleanCode, "def ")
		loopCount = strings.Count(cleanCode, "for ") + strings.Count(cleanCode, "while ")
	case "go":
		funcCount = strings.Count(cleanCode, "func ")
		loopCount = strings.Count(cleanCode, "for ") + strings.Count(cleanCode, "range ")
	case "javascript":
		funcCount = strings.Count(cleanCode, "function ") + strings.Count(cleanCode, "=>")
		loopCount = strings.Count(cleanCode, "for ") + strings.Count(cleanCode, "while ") + strings.Count(cleanCode, ".forEach")
	case "cpp":
		funcCount = strings.Count(cleanCode, "int main") + strings.Count(cleanCode, "void ")
		loopCount = strings.Count(cleanCode, "for ") + strings.Count(cleanCode, "while ")
	case "java":
		funcCount = strings.Count(cleanCode, "public static void main") + strings.Count(cleanCode, "class ")
		loopCount = strings.Count(cleanCode, "for ") + strings.Count(cleanCode, "while ")
	}

	// LAYER 4: NETWORK HEURISTICS
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Anti-Cheating: Normalized Code Hash (comments stripped, whitespace normalized)
	normalizedCode := strings.Join(strings.Fields(codeNoComments), "")
	codeHash := sha256.Sum256([]byte(normalizedCode))
	hashStr := hex.EncodeToString(codeHash[:])

	// Check for EXACT hash match cross-user
	var dupCount int64
	database.DB.Model(&models.Submission{}).
		Where("problem_id = ? AND code_hash = ? AND user_id != ?", problemID, hashStr, uid).
		Count(&dupCount)

	// Low-Trust User: Stricter Limits
	var user models.User
	database.DB.First(&user, "id = ?", uid)
	if user.TrustScore < 50 && subCount >= 10 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Low trust score: submission limit reduced to 10"})
		return
	}

	// Too-Fast Correct Submission Detection (flag if first submission is ACCEPTED in < 60s)
	var firstSub models.Submission
	database.DB.Where("user_id = ? AND problem_id = ?", uid, problemID).Order("created_at asc").First(&firstSub)
	tooFastFlag := false
	if firstSub.ID == "" {
		// This is the first submission, will check after execution if it's too fast
		tooFastFlag = true
	}

	submission := models.Submission{
		ID:        utils.GenerateID(),
		UserID:    uid,
		EventID:   problem.EventID,
		ProblemID: problemID,
		Code:      input.Code,
		Language:  input.Language,
		CodeHash:  hashStr,
		Status:    models.SubStatusPending, // Will update after execution
		CreatedAt: time.Now(),
	}

	// Save main submission
	if err := database.DB.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// Save Metrics
	metrics := models.SubmissionMetrics{
		SubmissionID:  submission.ID,
		PasteCount:    input.PasteCount,
		PastedChars:   input.PastedChars,
		BlurCount:     input.BlurCount,
		TabSwitchCnt:  input.BlurCount, // Map blur to tab switch for now
		IP:            clientIP,
		UserAgent:     userAgent,
		LineCount:     lineCount,
		FunctionCount: funcCount,
		LoopCount:     loopCount,
	}
	database.DB.Create(&metrics)

	// LAYER 2 & 4: FLAGGING & TRUST SCORE
	flags := []models.SubmissionFlag{}
	trustDeduction := 0

	// Flag: Duplicate Hash
	if dupCount > 0 {
		flags = append(flags, models.SubmissionFlag{
			ID:           utils.GenerateID(),
			SubmissionID: submission.ID,
			Type:         models.FlagTypeHash,
			Details:      fmt.Sprintf("Similar code hash found in %d other submissions", dupCount),
			CreatedAt:    time.Now(),
		})
		trustDeduction += 20
	}

	// Flag: Excessive Pasting (>50% of code is pasted)
	if len(cleanCode) > 50 && input.PastedChars > len(cleanCode)/2 {
		flags = append(flags, models.SubmissionFlag{
			ID:           utils.GenerateID(),
			SubmissionID: submission.ID,
			Type:         models.FlagTypePaste,
			Details:      fmt.Sprintf("Pasted %d chars (Total: %d)", input.PastedChars, len(cleanCode)),
			CreatedAt:    time.Now(),
		})
		trustDeduction += 10
	}

	// Flag: Excessive Focus Loss (>10 times)
	if input.BlurCount > 10 {
		flags = append(flags, models.SubmissionFlag{
			ID:           utils.GenerateID(),
			SubmissionID: submission.ID,
			Type:         models.FlagTypeBlur,
			Details:      fmt.Sprintf("Focus lost %d times", input.BlurCount),
			CreatedAt:    time.Now(),
		})
		trustDeduction += 15
	}

	// Flag: IP Sharing (Multiple users on same IP in same Event)
	var usersOnIP int64
	database.DB.Table("submission_metrics").
		Joins("JOIN submissions ON submissions.id = submission_metrics.submission_id").
		Where("submission_metrics.ip = ? AND submissions.event_id = ? AND submissions.user_id != ?", clientIP, problem.EventID, uid).
		Distinct("submissions.user_id").
		Count(&usersOnIP)

	if usersOnIP > 0 {
		flags = append(flags, models.SubmissionFlag{
			ID:           utils.GenerateID(),
			SubmissionID: submission.ID,
			Type:         models.FlagTypeSuspicious,
			Details:      fmt.Sprintf("IP shared with %d other users in this contest", usersOnIP),
			CreatedAt:    time.Now(),
		})
		trustDeduction += 30
	}

	// Flag: User-Agent Correlation (same UA + IP as another user)
	var sameUA int64
	database.DB.Table("submission_metrics").
		Joins("JOIN submissions ON submissions.id = submission_metrics.submission_id").
		Where("submission_metrics.ip = ? AND submission_metrics.user_agent = ? AND submissions.event_id = ? AND submissions.user_id != ?",
			clientIP, userAgent, problem.EventID, uid).
		Distinct("submissions.user_id").
		Count(&sameUA)

	if sameUA > 0 {
		flags = append(flags, models.SubmissionFlag{
			ID:           utils.GenerateID(),
			SubmissionID: submission.ID,
			Type:         models.FlagTypeSuspicious,
			Details:      fmt.Sprintf("Same IP + User-Agent as %d other users", sameUA),
			CreatedAt:    time.Now(),
		})
		trustDeduction += 25
	}

	// Store tooFastFlag for post-execution check
	_ = tooFastFlag // Will be used in execution callback

	// Apply Flags & Trust Score
	for _, f := range flags {
		database.DB.Create(&f)
	}

	if trustDeduction > 0 {
		// Decrease Trust Score (Cap at 0)
		database.DB.Model(&models.User{}).Where("id = ?", uid).
			Update("trust_score", gorm.Expr("GREATEST(trust_score - ?, 0)", trustDeduction))
	}

	// EXECUTION LOGIC (Piston)
	// Why loop? Because we have multiple test cases.
	// We run them sequentially.
	go func(subID string, prob models.Problem, userCode, lang string) {
		// Re-fetch submission to update later
		var sub models.Submission
		database.DB.First(&sub, "id = ?", subID)

		// Create execution context
		allPassed := true
		passedCases := 0
		totalExecTime := 0.0
		var lastRun *services.PistonExecuteResponse

		// Normalize output helper
		normalizeOutput := func(s string) string {
			s = strings.TrimSpace(s)
			s = strings.ReplaceAll(s, "\r\n", "\n")
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				lines[i] = strings.TrimRight(line, " \t")
			}
			return strings.Join(lines, "\n")
		}

		for _, tc := range prob.TestCases {
			start := time.Now()
			res, err := services.ExecuteCode(lang, userCode, tc.Input, prob.TimeLimit, prob.MemoryLimit)
			execDuration := time.Since(start).Seconds() * 1000 // ms
			totalExecTime += execDuration
			lastRun = res

			if err != nil {
				sub.Status = models.SubStatusRE // Runtime Error (or infra error)
				sub.Verdict = "Runtime Error"
				allPassed = false
				break
			}
			if res.Run.Signal == "SIGKILL" {
				sub.Status = models.SubStatusTLE
				sub.Verdict = "Time Limit Exceeded"
				allPassed = false
				break
			}
			if res.Run.Signal == "SIGTERM" {
				sub.Status = models.SubStatusTLE
				sub.Verdict = "Time Limit Exceeded"
				allPassed = false
				break
			}
			if res.Run.Signal != "" {
				sub.Status = models.SubStatusRE
				sub.Verdict = "Runtime Error (" + res.Run.Signal + ")"
				allPassed = false
				break
			}
			if res.Run.Code != 0 {
				sub.Status = models.SubStatusRE
				// Check for common exit codes
				if res.Run.Code == 137 { // 128 + 9 (SIGKILL)
					sub.Status = models.SubStatusTLE
					sub.Verdict = "Time Limit Exceeded"
				} else {
					sub.Verdict = "Runtime Error (Exit Code " + fmt.Sprintf("%d", res.Run.Code) + ")"
					// Capture stderr
					if len(res.Run.Stderr) > 0 {
						// truncate if too long
						msg := res.Run.Stderr
						if len(msg) > 100 {
							msg = msg[:100] + "..."
						}
						sub.Verdict += ": " + msg
					}
				}
				allPassed = false
				break
			}

			// Compare Output
			actual := normalizeOutput(res.Run.Stdout)
			expected := normalizeOutput(tc.Output)

			if actual != expected {
				sub.Status = models.SubStatusWA
				sub.Verdict = "Wrong Answer"
				allPassed = false
				break
			}
			passedCases++
		}

		if allPassed {
			sub.Status = models.SubStatusAC
			sub.Verdict = "Accepted"
		}

		sub.TestCasesPassed = passedCases
		sub.TotalTestCases = len(prob.TestCases)
		sub.Runtime = totalExecTime
		if lastRun != nil {
			// Convert Piston output to snapshot
			snap, _ := json.Marshal(lastRun)
			sub.OutputSnapshot = string(snap)
		}

		database.DB.Save(&sub)

		// Update Registration Score if AC
		if allPassed && prob.Points > 0 {
			var reg models.Registration
			if err := database.DB.Where("user_id = ? AND event_id = ?", sub.UserID, sub.EventID).First(&reg).Error; err == nil {
				// Check if already solved correctly before?
				// Simple MVP: Just add points if status wasn't AC before?
				// Better: Count unique solved problems.
				// For now, simpler: user gets points.
				// Wait, if they submit AC twice, do they get double points?
				// Need to check if they already have an AC submission for this problem.
				var count int64
				database.DB.Model(&models.Submission{}).Where("user_id = ? AND problem_id = ? AND status = ? AND id != ?", sub.UserID, sub.ProblemID, models.SubStatusAC, sub.ID).Count(&count)
				if count == 0 {
					reg.Score += prob.Points
					database.DB.Save(&reg)
				}
			}
		}

	}(submission.ID, problem, input.Code, input.Language)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Submission received",
		"submission": submission,
	})
}

// GetUserSubmissions returns all submissions for a user on a problem
func GetUserSubmissions(c *gin.Context) {
	problemID := c.Param("problemId")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var submissions []models.Submission
	if err := database.DB.Where("problem_id = ? AND user_id = ?", problemID, userID).
		Order("created_at desc").Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

// GetEventLeaderboard returns the leaderboard for an event
func GetEventLeaderboard(c *gin.Context) {
	eventID := c.Param("eventId")

	type LeaderboardEntry struct {
		UserID         string    `json:"userId"`
		Username       string    `json:"username"`
		Name           string    `json:"name"`
		TotalScore     int       `json:"totalScore"`
		SolvedCount    int       `json:"solvedCount"`
		TotalRuntime   float64   `json:"totalRuntime"`
		LastSubmitTime time.Time `json:"lastSubmitTime"`
	}

	// Logic: Get users registered for event, Sum points of Accepted Submissions (Distinct Problem best score).
	// Simplified MVP: Sum of all unique Accepted problems points.

	var entries []LeaderboardEntry
	// Raw SQL for complexity
	query := `
		SELECT 
			u.id as user_id, 
			u.username, 
			u.name,
			COALESCE(SUM(p.points), 0) as total_score,
			COUNT(DISTINCT s.problem_id) as solved_count,
			COALESCE(SUM(s.runtime), 0) as total_runtime,
			MAX(s.created_at) as last_submit_time
		FROM submissions s
		JOIN "User" u ON s.user_id = u.id
		JOIN problems p ON s.problem_id = p.id
		WHERE s.event_id = ? AND s.status = 'ACCEPTED'
		GROUP BY u.id, u.username, u.name
		ORDER BY total_score DESC, total_runtime ASC, last_submit_time ASC
	`

	database.DB.Raw(query, eventID).Scan(&entries)

	c.JSON(http.StatusOK, gin.H{"leaderboard": entries})
}
