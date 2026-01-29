package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"gorm.io/gorm"
)

// -- Inputs -- //

type RegisterEventInput struct {
	RazorpayPaymentID string `json:"razorpayPaymentId"`
	RazorpayOrderID   string `json:"razorpayOrderId"`
	RazorpaySignature string `json:"razorpaySignature"`
}

type CreateEventInput struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Slug        string  `json:"slug"`
	StartTime   string  `json:"startTime"` // ISO8601
	EndTime     string  `json:"endTime"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"` // UPCOMING, LIVE, ENDED
}

// -- Handlers -- //

// EventListResponse defines the enriched event structure
type EventListResponse struct {
	models.Event
	ProblemCount     int64                     `json:"problemCount"`
	ParticipantCount int64                     `json:"participantCount"`
	TotalPoints      int                       `json:"totalPoints"`
	IsRegistered     bool                      `json:"isRegistered"`
	UserStatus       models.RegistrationStatus `json:"userStatus,omitempty"`
}

// ListEvents handles GET /events
func ListEvents(c *gin.Context) {
	userId, exists := c.Get("userId")

	var events []models.Event
	// Show all for MVP, order by start time
	if result := database.DB.Preload("Problems").Order("start_time desc").Find(&events); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.ID
	}

	// Bulk Participant Counts
	type CountResult struct {
		EventID string
		Count   int64
	}
	var countResults []CountResult
	database.DB.Model(&models.Registration{}).Where("event_id IN ?", eventIDs).Select("event_id, count(*) as count").Group("event_id").Scan(&countResults)
	countMap := make(map[string]int64)
	for _, cr := range countResults {
		countMap[cr.EventID] = cr.Count
	}

	// Bulk User Registrations
	regMap := make(map[string]models.RegistrationStatus)
	if exists {
		var regs []models.Registration
		database.DB.Where("user_id = ? AND event_id IN ?", userId, eventIDs).Find(&regs)
		for _, reg := range regs {
			regMap[reg.EventID] = reg.Status
		}
	}

	var response []EventListResponse
	now := time.Now()

	for _, event := range events {
		// Lazy Status Update: Check if event should be ENDED or LIVE
		updated := false
		if event.Status != models.EventStatusEnded && now.After(event.EndTime) {
			event.Status = models.EventStatusEnded
			updated = true
		} else if event.Status == models.EventStatusUpcoming && now.After(event.StartTime) && now.Before(event.EndTime) {
			event.Status = models.EventStatusLive
			updated = true
		}

		if updated {
			database.DB.Model(&models.Event{}).Where("id = ?", event.ID).Update("status", event.Status)
		}

		// 1. Participant Count from map
		participantCount := countMap[event.ID]

		// 2. Count Problems & Points (Already preloaded)
		problemCount := int64(len(event.Problems))
		totalPoints := 0
		for _, p := range event.Problems {
			totalPoints += p.Points
		}

		// 3. User Registration from map
		userStatus, isRegistered := regMap[event.ID]

		safeEvent := event
		safeEvent.Problems = nil

		response = append(response, EventListResponse{
			Event:            safeEvent,
			ProblemCount:     problemCount,
			ParticipantCount: participantCount,
			TotalPoints:      totalPoints,
			IsRegistered:     isRegistered,
			UserStatus:       userStatus,
		})
	}

	c.Header("Cache-Control", "public, max-age=60")
	c.JSON(http.StatusOK, gin.H{"events": response})
}

// GetEvent handles GET /events/:id
func GetEvent(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userId") // Optional auth

	var event models.Event
	if result := database.DB.Preload("Problems").First(&event, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Lazy Status Update
	now := time.Now()
	updated := false
	if event.Status != models.EventStatusEnded && now.After(event.EndTime) {
		event.Status = models.EventStatusEnded
		updated = true
	} else if event.Status == models.EventStatusUpcoming && now.After(event.StartTime) && now.Before(event.EndTime) {
		event.Status = models.EventStatusLive
		updated = true
	}

	if updated {
		database.DB.Model(&event).Update("status", event.Status)
	}

	// Calculate Metadata
	problemCount := int64(len(event.Problems))
	totalPoints := 0
	for _, p := range event.Problems {
		totalPoints += p.Points
	}

	var participantCount int64
	database.DB.Model(&models.Registration{}).Where("event_id = ?", event.ID).Count(&participantCount)

	// Remove sensitive problem data
	event.Problems = nil

	// Check registration status if user logged in
	var isRegistered bool = false
	var rulesAccepted bool = false
	var regStatus models.RegistrationStatus = ""

	if userID != nil {
		var registration models.Registration
		if err := database.DB.Where("user_id = ? AND event_id = ?", userID, id).First(&registration).Error; err == nil {
			isRegistered = true
			rulesAccepted = registration.RulesAccepted
			regStatus = registration.Status
		}
	}

	// Security: Mask ExternalJoinURL unless it's the Join endpoint
	event.ExternalJoinURL = ""

	c.JSON(http.StatusOK, gin.H{
		"event": event,
		"metadata": gin.H{
			"problemCount":     problemCount,
			"totalPoints":      totalPoints,
			"participantCount": participantCount,
		},
		"isRegistered":       isRegistered,
		"rulesAccepted":      rulesAccepted,
		"registrationStatus": regStatus,
	})
}

// RegisterForEvent handles POST /events/:id/register
func RegisterForEvent(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var event models.Event
	if result := database.DB.First(&event, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Check existing
	var existing models.Registration
	if err := database.DB.Where("user_id = ? AND event_id = ?", userID, id).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already registered"})
		return
	}

	// Payment Logic
	status := models.RegStatusPaid // Default for free events
	if event.Price > 0 {
		var input RegisterEventInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment details required"})
			return
		}

		// --- Razorpay Verification ---
		secret := os.Getenv("RAZORPAY_KEY_SECRET")
		if secret == "" {
			// Fail securely if secret is missing in env
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment configuration error"})
			return
		}

		// Verify Signature
		data := input.RazorpayOrderID + "|" + input.RazorpayPaymentID
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(data))
		generatedSignature := hex.EncodeToString(h.Sum(nil))

		if generatedSignature != input.RazorpaySignature {
			c.JSON(http.StatusForbidden, gin.H{"error": "Payment verification failed: Invalid Signature"})
			return
		}
	}

	registration := models.Registration{
		ID:        utils.GenerateID(),
		UserID:    userID.(string),
		EventID:   id,
		Status:    status,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&registration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register for event"})
		return
	}

	// Increment User's Contest Count
	strUserID := userID.(string)
	database.DB.Model(&models.User{ID: strUserID}).Update("wrapped_contest_count", gorm.Expr("wrapped_contest_count + ?", 1))

	c.JSON(http.StatusOK, registration)
}

// AcceptRules handles POST /events/:id/rules (Acts as Join/Enter Contest)
func AcceptRules(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(string)

	// 1. Load Event
	var event models.Event
	if result := database.DB.First(&event, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// 2. Time Validation (Server Validity)
	// Practice Arena is always open
	if event.ID != "practice-arena-mvp" {
		now := time.Now()
		if now.Before(event.StartTime) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Contest has not started yet"})
			return
		}
		if now.After(event.EndTime) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Contest has ended"})
			return
		}
	}

	// 3. Find or Create Registration
	var registration models.Registration
	if err := database.DB.Where("user_id = ? AND event_id = ?", uid, id).First(&registration).Error; err != nil {
		// Not found -> Check eligibility to Auto-Join
		if event.Price > 0 {
			// Paid event: Cannot auto-join. Must use /register with payment.
			c.JSON(http.StatusForbidden, gin.H{"error": "Payment required to join this event"})
			return
		}

		// Free event: Auto-create registration
		registration = models.Registration{
			ID:              utils.GenerateID(),
			UserID:          uid,
			EventID:         id,
			Status:          models.RegStatusPaid, // Free = Paid/Confirmed
			RulesAccepted:   true,
			RulesAcceptedAt: time.Now(),
			CreatedAt:       time.Now(),
		}
		if err := database.DB.Create(&registration).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join event"})
			return
		}

		// Increment User's Contest Count
		database.DB.Model(&models.User{ID: uid}).Update("wrapped_contest_count", gorm.Expr("wrapped_contest_count + ?", 1))

		c.JSON(http.StatusOK, gin.H{"message": "Joined contest and accepted rules", "joined": true})
		return
	}

	// 4. Update existing registration
	if !registration.RulesAccepted {
		registration.RulesAccepted = true
		registration.RulesAcceptedAt = time.Now()
		database.DB.Save(&registration)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rules accepted", "joined": true})
}

// GetEventAccess handles GET /events/:id/access (The Gatekeeper)
func GetEventAccess(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var event models.Event
	if result := database.DB.First(&event, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// 1. Check Registration & Payment
	var registration models.Registration
	if err := database.DB.Where("user_id = ? AND event_id = ? AND status = ?", userID, id, models.RegStatusPaid).First(&registration).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access Denied: Payment required"})
		return
	}

	// 2. Check Time
	if time.Now().Before(event.StartTime) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Event has not started yet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access":  true,
		"message": "Access Granted",
	})
}

// CreateEvent (Admin)
func CreateEvent(c *gin.Context) {
	// Admin verification is handled by middleware
	var input CreateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime, _ := time.Parse(time.RFC3339, input.StartTime)
	endTime, _ := time.Parse(time.RFC3339, input.EndTime)

	// Default Slug if empty
	slug := input.Slug
	if slug == "" {
		slug = utils.GenerateID() // Helper likely exists or use UUID
	}

	// Get User ID from context (Admin)
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	event := models.Event{
		ID:          utils.GenerateID(),
		Title:       input.Title,
		Description: input.Description,
		Slug:        slug,
		StartTime:   startTime,
		EndTime:     endTime,
		Price:       input.Price,
		Status:      models.EventStatus(input.Status),
		CreatedBy:   userID.(string), // Set Creator
		CreatedAt:   time.Now(),
	}

	if event.Status == "" {
		event.Status = models.EventStatusUpcoming
	}

	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"event": event})
}

// JoinExternalContest handles POST /events/:id/join-external
func JoinExternalContest(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var event models.Event
	if result := database.DB.First(&event, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	if !event.IsExternal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This is not an external contest"})
		return
	}

	// 1. Validate Registration
	var registration models.Registration
	if err := database.DB.Where("user_id = ? AND event_id = ?", userID, id).First(&registration).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Registration required to join contest"})
		return
	}

	// 2. Validate Time Window
	now := time.Now()
	if now.Before(event.ExternalJoinVisibleAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Join link is not yet visible"})
		return
	}
	if now.After(event.EndTime) {
		c.JSON(http.StatusGone, gin.H{"error": "Contest has already ended"})
		return
	}

	// 3. Log Join Time
	nowJoined := time.Now()
	registration.JoinedExternalAt = &nowJoined
	registration.Status = models.RegStatusJoined
	database.DB.Save(&registration)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"url":      event.ExternalJoinURL,
		"joinedAt": nowJoined,
	})
}
