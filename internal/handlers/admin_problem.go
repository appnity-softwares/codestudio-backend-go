package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// --- Problem Management ---

// AdminGetProblem returns full problem details including hidden test cases and private fields
func AdminGetProblem(c *gin.Context) {
	problemID := c.Param("id")

	var problem models.Problem
	if err := database.DB.Preload("TestCases").First(&problem, "id = ?", problemID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Problem not found"})
		return
	}
	c.JSON(200, gin.H{"problem": problem})
}

func AdminCreateProblem(c *gin.Context) {
	adminID := getAdminID(c)
	var req struct {
		EventID     string  `json:"eventId" binding:"required"`
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		Difficulty  string  `json:"difficulty"`
		Points      int     `json:"points"`
		TimeLimit   float64 `json:"timeLimit"`
		MemoryLimit int     `json:"memoryLimit"`
		Penalty     int     `json:"penalty"`
		StarterCode string  `json:"starterCode"` // JSON string map[lang]code
		Order       int     `json:"order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	problem := models.Problem{
		ID:          uuid.New().String(),
		EventID:     req.EventID,
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		Points:      req.Points,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		Penalty:     req.Penalty,
		StarterCode: req.StarterCode,
		Order:       req.Order,
	}

	if err := database.DB.Create(&problem).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to create problem"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionCreateProblem, problem.ID, "problem", "Created Problem: "+problem.Title)

	c.JSON(201, gin.H{"problem": problem})
}

func AdminUpdateProblem(c *gin.Context) {
	problemID := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Difficulty  string  `json:"difficulty"`
		Points      int     `json:"points"`
		TimeLimit   float64 `json:"timeLimit"`
		MemoryLimit int     `json:"memoryLimit"`
		Penalty     int     `json:"penalty"`
		StarterCode string  `json:"starterCode"`
		Order       int     `json:"order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var problem models.Problem
		if err := tx.First(&problem, "id = ?", problemID).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{}
		if req.Title != "" {
			updates["title"] = req.Title
		}
		if req.Description != "" {
			updates["description"] = req.Description
		}
		if req.Difficulty != "" {
			updates["difficulty"] = req.Difficulty
		}
		updates["points"] = req.Points
		updates["time_limit"] = req.TimeLimit
		updates["memory_limit"] = req.MemoryLimit
		updates["penalty"] = req.Penalty
		if req.StarterCode != "" {
			updates["starter_code"] = req.StarterCode
		}
		updates["order"] = req.Order // Careful with 0, but acceptable

		if err := tx.Model(&problem).Updates(updates).Error; err != nil {
			return err
		}

		return logAdminAction(tx, adminID, models.ActionUpdateProblem, problemID, "problem", "Updated Problem")
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Problem Updated"})
}

func AdminDeleteProblem(c *gin.Context) {
	problemID := c.Param("id")
	adminID := getAdminID(c)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Check for submissions
		var count int64
		tx.Model(&models.Submission{}).Where("problem_id = ?", problemID).Count(&count)
		if count > 0 {
			return &gin.Error{Err: gorm.ErrInvalidTransaction, Type: gin.ErrorTypePublic, Meta: "Cannot delete problem with existing submissions"}
		}

		if err := tx.Delete(&models.Problem{}, "id = ?", problemID).Error; err != nil {
			return err
		}
		return logAdminAction(tx, adminID, models.ActionDeleteProblem, problemID, "problem", "Deleted Problem")
	})

	if err != nil {
		c.JSON(400, gin.H{"error": "Cannot delete problem"})
		return
	}
	c.JSON(200, gin.H{"message": "Problem Deleted"})
}

// AdminReorderProblems updates the order of problems in a contest
func AdminReorderProblems(c *gin.Context) {
	adminID := getAdminID(c)
	var req struct {
		EventID    string   `json:"eventId" binding:"required"`
		ProblemIDs []string `json:"problemIds" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for i, pid := range req.ProblemIDs {
			if err := tx.Model(&models.Problem{}).
				Where("id = ? AND event_id = ?", pid, req.EventID).
				Update("order", i+1).Error; err != nil {
				return err
			}
		}
		return logAdminAction(tx, adminID, models.ActionReorderProblems, req.EventID, "contest", "Reordered Problems")
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Problems reordered"})
}

// --- Test Case Management ---

func AdminCreateTestCase(c *gin.Context) {
	problemID := c.Param("id") // Problem ID from URL
	adminID := getAdminID(c)

	var req struct {
		Input    string `json:"input" binding:"required"`
		Output   string `json:"output" binding:"required"`
		IsHidden bool   `json:"isHidden"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tc := models.TestCase{
		ID:        uuid.New().String(),
		ProblemID: problemID,
		Input:     req.Input,
		Output:    req.Output,
		IsHidden:  req.IsHidden,
	}

	if err := database.DB.Create(&tc).Error; err != nil {
		c.JSON(500, gin.H{"error": "DB Error"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionUpdateProblem, problemID, "testcase", "Added Test Case")
	c.JSON(201, gin.H{"testCase": tc})
}

func AdminUpdateTestCase(c *gin.Context) {
	tcID := c.Param("tcId")
	adminID := getAdminID(c)

	var req struct {
		Input    string `json:"input"`
		Output   string `json:"output"`
		IsHidden *bool  `json:"isHidden"` // Pointer to handle false
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Input != "" {
		updates["input"] = req.Input
	}
	if req.Output != "" {
		updates["output"] = req.Output
	}
	if req.IsHidden != nil {
		updates["is_hidden"] = *req.IsHidden
	}

	if err := database.DB.Model(&models.TestCase{}).Where("id = ?", tcID).Updates(updates).Error; err != nil {
		c.JSON(500, gin.H{"error": "DB Error"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionUpdateProblem, tcID, "testcase", "Updated Test Case")
	c.JSON(200, gin.H{"message": "Test Case Updated"})
}

func AdminDeleteTestCase(c *gin.Context) {
	tcID := c.Param("tcId")
	adminID := getAdminID(c)

	if err := database.DB.Delete(&models.TestCase{}, "id = ?", tcID).Error; err != nil {
		c.JSON(500, gin.H{"error": "DB Error"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionUpdateProblem, tcID, "testcase", "Deleted Test Case")
	c.JSON(200, gin.H{"message": "Test Case Deleted"})
}
