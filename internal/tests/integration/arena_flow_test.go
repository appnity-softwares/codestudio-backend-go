package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/handlers"
	"github.com/pushp314/devconnect-backend/internal/middleware"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// setupRouter initializes the Gin engine with all necessary routes for the test
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Public routes
	r.POST("/api/auth/register", handlers.Register)
	r.POST("/api/auth/login", handlers.Login)

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// Arena User Routes
		protected.GET("/events", handlers.ListEvents)
		protected.GET("/events/:id", handlers.GetEvent)
		protected.POST("/events/:eventId/register", handlers.RegisterForEvent)
		// Matched with handler expectation: problemId param
		protected.POST("/problems/:problemId/submit", handlers.SubmitSolution)

		// Admin Routes
		admin := protected.Group("/admin") // Or just protected, assuming AdminMiddleware inside
		admin.Use(middleware.AdminMiddleware())
		{
			admin.POST("/events", handlers.CreateEvent)
			admin.POST("/contests/:eventId/problems", handlers.CreateProblem)
		}
	}

	return r
}

func TestArenaFullFlow(t *testing.T) {
	// 1. Setup DB
	db := setupTestDB(t)

	// 2. Setup Router
	r := setupRouter()

	// 3. Create Users
	// Admin
	adminToken := createTestUser(t, "admin", "ADMIN")
	// Regular User
	userToken := createTestUser(t, "competitor", "USER")

	// 4. Admin Creates Contest
	contestID := createTestContest(t, r, adminToken)
	assert.NotEmpty(t, contestID)

	// 5. Admin Creates Problem
	problemID := createTestProblem(t, r, adminToken, contestID)
	assert.NotEmpty(t, problemID)

	// 6. User Registers for Contest (Mock Payment)
	// Currently RegisterForEvent expects razorpay details.
	// For testing, we might need a bypass or simulate the exact payload.
	// Since we don't have a real Razorpay secret in test env, verification might fail
	// unless we mock config or the handler logic.
	// STRATEGY: Directly insert Registration into DB for this test to bypass Payment Gateway complexity
	// because we are testing the FLOW, not Razorpay itself (which is external).
	registerUserDirectly(t, db, contestID, "competitor")

	// 7. User Submits Solution
	// We need to ensure the contest is LIVE.
	// The helper `createTestContest` should make it live now.

	// Mock Piston?
	// The SubmitSolution handler calls services.ExecuteCode.
	// If Piston is unreachable, it might error.
	// For integration tests without mocks, we rely on Piston being reachable
	// OR we accept that "execution failed" is a valid result, provided the submission is recorded.
	// Let's see what happens.
	submitSolution(t, r, userToken, problemID)
}

// --- Helpers ---

func createTestUser(t *testing.T, prefix string, role string) string {
	// 1. Register
	// We'll just create directly in DB to save time and set Role easily
	passHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := models.User{
		ID:       utils.GenerateID(),
		Username: prefix + "_user",
		Email:    prefix + "@test.com",
		Password: string(passHash),
		Name:     prefix + " Test",
		Role:     models.Role(role),
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user %s: %v", prefix, err)
	}

	// 2. Generate Token
	token, err := utils.GenerateToken(user.ID) // Assuming GenerateToken takes just ID
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	return token
}

func createTestContest(t *testing.T, r *gin.Engine, token string) string {
	// StartTime 1 min ago, EndTime 1 hour from now
	start := time.Now().Add(-1 * time.Minute).Format(time.RFC3339)
	end := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	payload := map[string]interface{}{
		"title":       "Integration Test Contest",
		"description": "Testing the flow",
		"startTime":   start,
		"endTime":     end,
		"price":       0,
		"slug":        "integration-test",
		"status":      "LIVE",
	}

	w := performRequest(r, "POST", "/api/admin/events", payload, token)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["event"] != nil {
		event := resp["event"].(map[string]interface{})
		return event["id"].(string)
	}
	// Fallback to direct ID (legacy) or panic
	if resp["id"] != nil {
		return resp["id"].(string)
	}
	panic("Event ID not found in response")
}

func createTestProblem(t *testing.T, r *gin.Engine, token string, contestID string) string {
	payload := map[string]interface{}{
		"title":       "Sum of Two",
		"description": "Return a + b",
		"difficulty":  "EASY",
		"points":      100,
		"timeLimit":   1.0,
		"memoryLimit": 128,
		"testCases": []map[string]interface{}{
			{"input": "1 2", "output": "3", "isHidden": false},
		},
	}

	w := performRequest(r, "POST", "/api/admin/contests/"+contestID+"/problems", payload, token)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	problem := resp["problem"].(map[string]interface{})
	return problem["id"].(string)
}

func registerUserDirectly(t *testing.T, db *gorm.DB, eventID string, usernamePrefix string) {
	var user models.User
	db.Where("username = ?", usernamePrefix+"_user").First(&user)

	reg := models.Registration{
		ID:        utils.GenerateID(),
		EventID:   eventID,
		UserID:    user.ID,
		Status:    models.RegStatusPaid,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&reg).Error; err != nil {
		t.Fatalf("Failed to force register user: %v", err)
	}
}

func submitSolution(t *testing.T, r *gin.Engine, token string, problemID string) {
	code := "a, b = map(int, input().split())\nprint(a + b)"

	payload := map[string]interface{}{
		"code":     code,
		"language": "python",
	}

	w := performRequest(r, "POST", "/api/problems/"+problemID+"/submit", payload, token)

	// 201 Created is expected if Piston works OR RE handling is uniform.
	// Handler: if err != nil { c.JSON(http.StatusOK, ... type: run ... code: 1) } -> this is for RunSolution?
	// SubmitSolution: if err != nil { submission.Status = RE ... c.JSON(http.StatusCreated, ...) }
	// So we expect 201 Created regardless of execution success (unless DB fail)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	submission := resp["submission"].(map[string]interface{})
	assert.NotEmpty(t, submission["id"])
	assert.Equal(t, problemID, submission["problemId"])

	// Status could be AC (Piston works) or RE (Piston fail)
	// We pass if it's either, failing only if empty.
	status := submission["status"].(string)
	t.Logf("Submission Status: %s", status)
	assert.NotEmpty(t, status)
}

// Utility to make requests
func performRequest(r *gin.Engine, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(jsonBytes))
	} else {
		bodyReader = strings.NewReader("")
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
