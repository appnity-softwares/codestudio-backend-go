package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"gorm.io/gorm"
)

// --- Practice Problem Management (Admin) ---

// AdminListPracticeProblems returns a list of all practice problems
func AdminListPracticeProblems(c *gin.Context) {
	problems := []models.PracticeProblem{}
	// Order by most recent first, using the exact column casing from DB
	if err := database.DB.Order("\"createdAt\" desc").Find(&problems).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch problems: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"problems": problems})
}

// AdminGetPracticeProblem returns details of a single practice problem
func AdminGetPracticeProblem(c *gin.Context) {
	id := c.Param("id")
	var problem models.PracticeProblem
	if err := database.DB.First(&problem, "id = ?", id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Problem not found"})
		return
	}
	c.JSON(200, gin.H{"problem": problem})
}

// AdminCreatePracticeProblem creates a new practice problem
func AdminCreatePracticeProblem(c *gin.Context) {
	adminID := getAdminID(c)
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Difficulty  string `json:"difficulty"` // EASY, MEDIUM, HARD
		Category    string `json:"category"`
		StarterCode string `json:"starterCode"`
		Solution    string `json:"solutionCode"`
		TestCases   string `json:"testCases"` // JSON string
		Language    string `json:"language"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	problem := models.PracticeProblem{
		ID:           uuid.New().String(),
		Title:        req.Title,
		Description:  req.Description,
		Difficulty:   req.Difficulty,
		Category:     req.Category,
		StarterCode:  req.StarterCode,
		SolutionCode: req.Solution, // Only admins see this
		TestCases:    req.TestCases,
		Language:     req.Language,
		TimeLimit:    2.0, // Default
		MemoryLimit:  128, // Default
		CreatorID:    adminID,
		CreatedAt:    time.Now(),
	}

	if err := database.DB.Create(&problem).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to create practice problem: " + err.Error()})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionCreateProblem, problem.ID, "practice_problem", "Created: "+problem.Title)
	c.JSON(201, gin.H{"problem": problem})
}

// AdminUpdatePracticeProblem updates an existing practice problem
func AdminUpdatePracticeProblem(c *gin.Context) {
	id := c.Param("id")
	adminID := getAdminID(c)

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Difficulty  string `json:"difficulty"`
		Category    string `json:"category"`
		StarterCode string `json:"starterCode"`
		Solution    string `json:"solutionCode"`
		TestCases   string `json:"testCases"`
		Language    string `json:"language"`
		IsDaily     *bool  `json:"isDailyProblem"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var problem models.PracticeProblem
		if err := tx.First(&problem, "id = ?", id).Error; err != nil {
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
		if req.Category != "" {
			updates["category"] = req.Category
		}
		if req.StarterCode != "" {
			updates["starter_code"] = req.StarterCode
		}
		if req.Solution != "" {
			updates["solution_code"] = req.Solution
		}
		if req.TestCases != "" {
			updates["test_cases"] = req.TestCases
		}
		if req.Language != "" {
			updates["language"] = req.Language
		}
		if req.IsDaily != nil {
			updates["is_daily_problem"] = *req.IsDaily
		}

		if err := tx.Model(&problem).Updates(updates).Error; err != nil {
			return err
		}

		return logAdminAction(tx, adminID, models.ActionUpdateProblem, id, "practice_problem", "Updated Practice Problem")
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Problem updated"})
}

// AdminDeletePracticeProblem deletes a practice problem
func AdminDeletePracticeProblem(c *gin.Context) {
	id := c.Param("id")
	adminID := getAdminID(c)

	if err := database.DB.Delete(&models.PracticeProblem{}, "id = ?", id).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete problem"})
		return
	}

	logAdminAction(database.DB, adminID, models.ActionDeleteProblem, id, "practice_problem", "Deleted Practice Problem")
	c.JSON(200, gin.H{"message": "Problem deleted"})
}
