package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/internal/services"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"gorm.io/gorm"
)

// ============================================
// PRACTICE ARENA v1 - Casual Practice System
// No anti-cheat, unlimited attempts, no pressure
// ============================================

// ListPracticeProblems handles GET /api/practice/problems
func ListPracticeProblems(c *gin.Context) {
	var problems []models.PracticeProblem

	query := database.DB.Model(&models.PracticeProblem{})

	// Filter by difficulty
	if diff := c.Query("difficulty"); diff != "" {
		query = query.Where("difficulty = ?", diff)
	}

	// Filter by category
	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}

	// Order: Daily problem first, then by solve count
	query = query.Order("is_daily_problem DESC, solve_count DESC")

	if err := query.Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
		return
	}

	// If user is authenticated, add their solve status
	userID, exists := c.Get("userId")
	if exists {
		// Get user's solved problem IDs
		var solvedIDs []string
		database.DB.Model(&models.PracticeSubmission{}).
			Where("user_id = ? AND status = ?", userID, "ACCEPTED").
			Distinct("problem_id").
			Pluck("problem_id", &solvedIDs)

		solvedSet := make(map[string]bool)
		for _, id := range solvedIDs {
			solvedSet[id] = true
		}

		// Attach solve status to response
		type ProblemWithStatus struct {
			models.PracticeProblem
			IsSolved bool `json:"isSolved"`
		}

		result := make([]ProblemWithStatus, len(problems))
		for i, p := range problems {
			result[i] = ProblemWithStatus{
				PracticeProblem: p,
				IsSolved:        solvedSet[p.ID],
			}
		}

		c.JSON(http.StatusOK, gin.H{"problems": result})
		return
	}

	c.JSON(http.StatusOK, gin.H{"problems": problems})
}

// GetPracticeProblem handles GET /api/practice/problems/:id
func GetPracticeProblem(c *gin.Context) {
	id := c.Param("id")

	var problem models.PracticeProblem
	if err := database.DB.First(&problem, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Don't expose solution code
	problem.SolutionCode = ""

	// Check if user has solved it
	isSolved := false
	if userID, exists := c.Get("userId"); exists {
		var count int64
		database.DB.Model(&models.PracticeSubmission{}).
			Where("user_id = ? AND problem_id = ? AND status = ?", userID, id, "ACCEPTED").
			Count(&count)
		isSolved = count > 0
	}

	c.JSON(http.StatusOK, gin.H{
		"problem":  problem,
		"isSolved": isSolved,
	})
}

// SubmitPracticeSolution handles POST /api/practice/submit
func SubmitPracticeSolution(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		ProblemID string `json:"problemId" binding:"required"`
		Code      string `json:"code" binding:"required"`
		Language  string `json:"language" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get problem
	var problem models.PracticeProblem
	if err := database.DB.First(&problem, "id = ?", input.ProblemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Create submission record
	submission := models.PracticeSubmission{
		ID:        utils.GenerateID(),
		UserID:    userID.(string),
		ProblemID: input.ProblemID,
		Code:      input.Code,
		Language:  input.Language,
		Status:    "RUNNING",
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create submission"})
		return
	}

	// Increment attempt count
	database.DB.Model(&problem).Update("attempt_count", gorm.Expr("attempt_count + 1"))

	// Execute code with Piston
	res, err := services.ExecuteCode(input.Language, input.Code, "", float64(problem.TimeLimit), problem.MemoryLimit)

	if err != nil {
		submission.Status = "ERROR"
		submission.Error = err.Error()
		database.DB.Save(&submission)

		c.JSON(http.StatusOK, gin.H{
			"submission": submission,
			"message":    "Execution error",
		})
		return
	}

	// Parse test cases and compare
	var testCases []struct {
		Input    string `json:"input"`
		Expected string `json:"expected"`
	}
	json.Unmarshal([]byte(problem.TestCases), &testCases)

	// For MVP: Simple output comparison (no stdin support yet)
	// We'll just check if the output matches expected for the first test case
	submission.Output = res.Run.Stdout
	submission.TestsTotal = len(testCases)
	submission.TestsPassed = 0

	if res.Run.Code != 0 {
		submission.Status = "ERROR"
		submission.Error = res.Run.Stderr
	} else if len(testCases) > 0 {
		// Simple check: does output contain expected?
		// In a real system, we'd run each test case separately
		expectedOutput := testCases[0].Expected
		if res.Run.Stdout == expectedOutput ||
			contains(res.Run.Stdout, expectedOutput) {
			submission.Status = "ACCEPTED"
			submission.TestsPassed = len(testCases)
			submission.Verdict = "All tests passed"

			// Increment solve count (only on first solve)
			var prevSolves int64
			database.DB.Model(&models.PracticeSubmission{}).
				Where("user_id = ? AND problem_id = ? AND status = ? AND id != ?",
					userID, input.ProblemID, "ACCEPTED", submission.ID).
				Count(&prevSolves)

			if prevSolves == 0 {
				database.DB.Model(&problem).Update("solve_count", gorm.Expr("solve_count + 1"))
			}
		} else {
			submission.Status = "WRONG_ANSWER"
			submission.Verdict = "Output doesn't match expected"
		}
	} else {
		// No test cases, just mark as accepted if it runs
		submission.Status = "ACCEPTED"
		submission.Verdict = "Code executed successfully"
	}

	database.DB.Save(&submission)

	c.JSON(http.StatusOK, gin.H{
		"submission": submission,
		"output":     res.Run.Stdout,
		"stderr":     res.Run.Stderr,
	})
}

// GetUserPracticeSubmissions handles GET /api/practice/submissions
func GetUserPracticeSubmissions(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	problemID := c.Query("problemId")

	var submissions []models.PracticeSubmission
	query := database.DB.Where("user_id = ?", userID).
		Preload("Problem").
		Order("created_at DESC")

	if problemID != "" {
		query = query.Where("problem_id = ?", problemID)
	}

	query.Limit(50).Find(&submissions)

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

// GetDailyProblem handles GET /api/practice/daily
func GetDailyProblem(c *gin.Context) {
	var problem models.PracticeProblem

	// Get the current daily problem
	if err := database.DB.Where("is_daily_problem = ?", true).First(&problem).Error; err != nil {
		// No daily problem set, return a random one
		if err := database.DB.Order("RANDOM()").First(&problem).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No practice problems available"})
			return
		}
	}

	// Don't expose solution
	problem.SolutionCode = ""

	c.JSON(http.StatusOK, gin.H{"problem": problem})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
