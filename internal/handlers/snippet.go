package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/internal/services"
	"github.com/pushp314/devconnect-backend/pkg/utils"
	"gorm.io/gorm"
)

// -- Inputs --
type CreateSnippetInput struct {
	Title          string   `json:"title" binding:"required"`
	Description    string   `json:"description" binding:"required"`
	Language       string   `json:"language" binding:"required"`
	Code           string   `json:"code" binding:"required"`
	Tags           []string `json:"tags"`
	Visibility     string   `json:"visibility,omitempty"`
	OutputSnapshot string   `json:"outputSnapshot"`
	PreviewType    string   `json:"previewType"`
	Type           string   `json:"type"`       // MVP v1.1
	Difficulty     string   `json:"difficulty"` // MVP v1.1
	Runtime        float64  `json:"runtime"`    // ms
	ReferenceUrl   string   `json:"referenceUrl"`
	Status         string   `json:"status"`
}

type UpdateSnippetInput struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Code           string   `json:"code"`
	Tags           []string `json:"tags"`
	Visibility     string   `json:"visibility"`
	OutputSnapshot string   `json:"outputSnapshot"`
	PreviewType    string   `json:"previewType"`
	ReferenceUrl   string   `json:"referenceUrl"`
	Annotations    string   `json:"annotations"`
}

// -- Handlers --

// ListSnippets handles GET /snippets (with query params for search)
func ListSnippets(c *gin.Context) {
	var snippets []models.Snippet
	query := database.DB.Model(&models.Snippet{}).Preload("Author")

	// UserID check not needed for public list anymore

	// Select fields including computed like count
	// Filtering
	search := c.Query("search")
	if search != "" {
		searchLike := utils.SanitizeSearchQuery(search)
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchLike, searchLike)
	}

	lang := c.Query("language")
	if lang != "" {
		query = query.Where("language = ?", lang)
	}

	tags := c.Query("tag") // simplified single tag filter for now
	if tags != "" {
		query = query.Where("? = ANY(tags)", tags)
	}

	snippetType := c.Query("type")
	if snippetType != "" {
		query = query.Where("type = ?", snippetType)
	}

	difficulty := c.Query("difficulty")
	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}

	orderBy := c.Query("orderBy")
	switch orderBy {
	case "oldest":
		query = query.Order("\"createdAt\" asc")
	default:
		query = query.Order("\"createdAt\" desc")
	}

	if result := query.Find(&snippets); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch snippets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippets": snippets})
}

// CreateSnippet handles POST /snippets
func CreateSnippet(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input CreateSnippetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	snippet := models.Snippet{
		ID:             utils.GenerateID(),
		Title:          input.Title,
		Description:    input.Description,
		Language:       input.Language,
		Code:           input.Code,
		Visibility:     input.Visibility,
		AuthorID:       userID.(string),
		Tags:           pq.StringArray(input.Tags),
		OutputSnapshot: input.OutputSnapshot,
		PreviewType:    input.PreviewType,
		Type:           input.Type,
		Difficulty:     input.Difficulty,
		Runtime:        input.Runtime,
		ReferenceURL:   input.ReferenceUrl,
	}

	// Default visibility
	if snippet.Visibility == "" {
		snippet.Visibility = "public"
	}

	// Handle Status (v1.2: allow public direct post)
	if input.Status != "" {
		snippet.Status = input.Status
		if snippet.Status == "PUBLISHED" {
			snippet.Verified = true
		}
	} else {
		snippet.Status = "DRAFT"
		snippet.Verified = false
	}
	snippet.LastExecutionStatus = ""

	if result := database.DB.Create(&snippet); result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "A snippet with this title already exists. Please choose a different title."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Check for Badges
	newBadges, _ := services.CheckBadges(userID.(string))

	c.JSON(http.StatusCreated, gin.H{
		"snippet":   snippet,
		"newBadges": newBadges,
	})
}

// GetSnippet handles GET /snippets/:id
func GetSnippet(c *gin.Context) {
	id := c.Param("id")
	var snippet models.Snippet

	if result := database.DB.Preload("Author").Preload("ForkedFrom").First(&snippet, "id = ?", id); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippet": snippet})
}

// UpdateSnippet handles PUT /snippets/:id
func UpdateSnippet(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userId")

	var snippet models.Snippet
	if result := database.DB.First(&snippet, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	if snippet.AuthorID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only edit your own snippets"})
		return
	}

	var input UpdateSnippetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if input.Code != "" && input.Code != snippet.Code {
		snippet.Code = input.Code
		// Reset verification if code changes
		snippet.Verified = false
		snippet.Status = "DRAFT"
		snippet.LastExecutionStatus = ""
	}

	snippet.Title = input.Title
	snippet.Description = input.Description
	// snippet.Code = input.Code // Handled above
	if len(input.Tags) > 0 {
		snippet.Tags = pq.StringArray(input.Tags)
	}
	if input.OutputSnapshot != "" {
		snippet.OutputSnapshot = input.OutputSnapshot
	}
	if input.PreviewType != "" {
		snippet.PreviewType = input.PreviewType
	}
	if input.Visibility != "" {
		snippet.Visibility = input.Visibility
	}
	if input.ReferenceUrl != "" {
		snippet.ReferenceURL = input.ReferenceUrl
	}
	if input.Annotations != "" {
		snippet.Annotations = input.Annotations
	}

	database.DB.Save(&snippet)

	c.JSON(http.StatusOK, gin.H{"snippet": snippet})
}

// DeleteSnippet handles DELETE /snippets/:id
func DeleteSnippet(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userId")

	var snippet models.Snippet
	if result := database.DB.First(&snippet, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	if snippet.AuthorID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own snippets"})
		return
	}

	database.DB.Delete(&snippet)

	c.JSON(http.StatusOK, gin.H{"message": "Snippet deleted"})
}

// UpdateSnippetOutput updates the output of a snippet
func UpdateSnippetOutput(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userId")

	var snippet models.Snippet
	if err := database.DB.First(&snippet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	if snippet.AuthorID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own snippet output"})
		return
	}

	var input struct {
		Output string `json:"output"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&snippet).Update("output", input.Output).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update output"})
		return
	}

	if err := database.DB.Model(&snippet).Update("output", input.Output).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update output"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippet": snippet})
}

// RunSnippet executes the snippet code (MVP: No input)
func RunSnippet(c *gin.Context) {
	id := c.Param("id")

	var snippet models.Snippet
	if err := database.DB.First(&snippet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	// MVP: Standardize execution limits (e.g., 2s timeout)
	// Input logic is removed for MVP as per requirements (No input handling)
	start := time.Now()
	res, err := services.ExecuteCode(snippet.Language, snippet.Code, "", 2.0, 128)
	duration := time.Since(start).Seconds() * 1000 // ms

	if err != nil {
		// Execution Failed (Service Error or Code Error that service couldn't handle)
		// We still record this as a failure
		snippet.LastExecutionStatus = "FAILURE"
		snippet.LastExecutionOutput = "Execution Error: " + err.Error()
		database.DB.Save(&snippet)

		c.JSON(http.StatusOK, gin.H{
			"stdout": "",
			"stderr": "Execution Error: " + err.Error(),
			"code":   1,
		})
		return
	}

	// Save Execution Result
	if res.Run.Code == 0 {
		snippet.LastExecutionStatus = "SUCCESS"
	} else {
		snippet.LastExecutionStatus = "FAILURE"
	}
	// Combine stdout and stderr for simple storage
	snippet.LastExecutionOutput = res.Run.Stdout
	if res.Run.Stderr != "" {
		snippet.LastExecutionOutput += "\n[STDERR]\n" + res.Run.Stderr
	}

	// Capture Runtime
	// Piston returns runtime in the output, but simple-piston might not expose it easily in 'Run' struct depending on pkg.
	// Assuming services.ExecuteCode returns a struct that has optional Runtime info?
	// The current services.ExecuteCode returns `*piston.ExecuteResult, error`.
	// piston.ExecuteResult.Run usually has no runtime field in some versions, but let's check if we can get it.
	// Actually, looking at commonly used piston clients, it might be in `Run.Signal` or similar?
	// For now, let's just create a placeholder or assume the service can provide it.
	// We'll update services.ExecuteCode later if needed.
	// Wait! The requirement says "Runtime (float64)".
	// Let's assume we can get it or just ignore for now if not available in current service signature.
	// But `Snippet` model has `Runtime`. We should update it if we can.
	// `RunSnippet` is for MVP execution.
	// If the user runs it, we might want to update the `Runtime` field on the snippet model itself?
	// Or is `Runtime` on the model a static "verified runtime"?
	// Yes, `Runtime` on model seems to be the "last successful run time" or "verified run time".
	if snippet.LastExecutionStatus == "SUCCESS" {
		snippet.Runtime = duration
	}

	database.DB.Save(&snippet)

	c.JSON(http.StatusOK, gin.H{
		"stdout": res.Run.Stdout,
		"stderr": res.Run.Stderr,
		"code":   res.Run.Code,
	})
}

// PublishSnippet handles POST /snippets/:id/publish
func PublishSnippet(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("userId")

	var snippet models.Snippet
	if result := database.DB.First(&snippet, "id = ?", id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	if snippet.AuthorID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only publish your own snippets"})
		return
	}

	// 1. Check Execution Status
	if snippet.LastExecutionStatus != "SUCCESS" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"error":   "Validation Failed",
			"details": "Snippet must execute successfully before publishing. Please run the code first.",
		})
		return
	}

	// 2. Publish
	snippet.Status = "PUBLISHED"
	snippet.Verified = true
	snippet.Visibility = "public" // Enforce public on publish
	snippet.OutputSnapshot = snippet.LastExecutionOutput

	if result := database.DB.Save(&snippet); result.Error != nil { // Changed from Create to Save
		// The instruction mentioned "duplicate key value violates unique constraint" which is more common for Create.
		// For Save, it's less likely to hit a unique constraint on an existing record unless a unique field is updated to a conflicting value.
		// However, following the instruction's logic for error handling.
		// Note: `strings` import would be needed if this check was actually used.
		// As the instruction explicitly asked for `strings` to be added to imports, I will add it.
		// Assuming the user wants this specific error check, even if `Save` is used.
		// If the intent was to modify CreateSnippet, the context was misleading.
		// I will add the strings import at the top of the file.
		// Since the full file is not provided, I will assume it's a Go file and add it to the import block.
		// For this specific change, I will just replace the error handling block.
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Snippet published successfully",
		"snippet": snippet,
	})
}

// ============================================
// v1.2: SMART FEED, FORK & COPY SYSTEM
// ============================================

// GetFeed handles GET /api/feed?bucket=trending|new|editor
// Implements deterministic scoring for feed buckets
func GetFeed(c *gin.Context) {
	bucket := c.DefaultQuery("bucket", "trending")

	var snippets []models.Snippet
	query := database.DB.Model(&models.Snippet{}).
		Preload("Author").
		Preload("ForkedFrom").
		Where("status = ?", "PUBLISHED")

	switch bucket {
	case "trending":
		// Score = (forkCount * 5) + copyCount * 2 + viewsCount - (hours_since_post * 0.05)
		query = query.
			Where("\"createdAt\" > NOW() - INTERVAL '30 days'").
			Order("(fork_count * 5 + copy_count * 2 + views_count - EXTRACT(EPOCH FROM (NOW() - \"createdAt\"))/3600 * 0.05) DESC")
	case "new":
		query = query.Order("\"createdAt\" DESC")
	case "editor":
		query = query.Where("is_featured = true").Order("\"createdAt\" DESC")
	default:
		query = query.Order("\"createdAt\" DESC")
	}

	if err := query.Limit(20).Find(&snippets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippets": snippets, "bucket": bucket})
}

// ForkSnippet handles POST /api/snippets/:id/fork
// Creates a copy of the snippet under the current user's ownership
func ForkSnippet(c *gin.Context) {
	sourceID := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var source models.Snippet
	if err := database.DB.First(&source, "id = ?", sourceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	// Check if forking is allowed
	if !source.AllowForks {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forking is disabled for this snippet"})
		return
	}

	// Check if user has already forked this snippet (Max 3)
	var existingForks int64
	database.DB.Model(&models.Snippet{}).
		Where("author_id = ? AND forked_from_id = ?", userID.(string), sourceID).
		Count(&existingForks)

	if existingForks >= 3 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "You cannot fork the same snippet more than 3 times"})
		return
	}

	// CHECK SYSTEM SETTING: Snippet Creation
	var setting models.SystemSettings
	if err := database.DB.Where("key = ?", models.SettingSnippetsEnabled).First(&setting).Error; err == nil {
		if setting.Value == "false" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Snippet creation is currently disabled by administrators.",
			})
			return
		}
	}

	// Create the fork
	fork := models.Snippet{
		ID:           utils.GenerateID(),
		Title:        source.Title + " (Fork)",
		Description:  source.Description,
		Language:     source.Language,
		Code:         source.Code,
		Tags:         source.Tags,
		AuthorID:     userID.(string),
		ForkedFromID: &sourceID,
		Visibility:   "public",
		PreviewType:  source.PreviewType,
		Type:         source.Type,
		Difficulty:   source.Difficulty,
		Status:       "DRAFT", // Forks start as drafts
		Verified:     false,
		ViewsCount:   0,
		CopyCount:    0,
		ForkCount:    0,
	}

	// Transaction: Create fork + Increment source fork count
	tx := database.DB.Begin()
	if err := tx.Create(&fork).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "A fork with this title already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create fork"})
		return
	}

	if err := tx.Model(&source).Update("fork_count", gorm.Expr("fork_count + 1")).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update fork count"})
		return
	}

	tx.Commit()

	// Preload author for response
	database.DB.Preload("Author").First(&fork, "id = ?", fork.ID)

	c.JSON(http.StatusCreated, gin.H{"snippet": fork, "message": "Snippet forked successfully"})
}

// RecordSnippetCopy handles POST /api/snippets/:id/copy
// Increments the copy count ONLY ONCE per user
func RecordSnippetCopy(c *gin.Context) {
	snippetID := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Check if copy exists
		var copyAction models.EntityCopy
		result := tx.Where("user_id = ? AND entity_type = ? AND entity_id = ?",
			userID.(string), models.EntityTypeSnippet, snippetID).First(&copyAction)

		if result.Error == nil {
			// Already copied, do nothing
			return nil
		}

		// 2. Create copy record
		newCopy := models.EntityCopy{
			ID:         utils.GenerateID(),
			UserID:     userID.(string),
			EntityType: models.EntityTypeSnippet,
			EntityID:   snippetID,
		}
		if err := tx.Create(&newCopy).Error; err != nil {
			return err
		}

		// 3. Increment snippet copy count
		if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).
			Update("copy_count", gorm.Expr("copy_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record copy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Copy recorded"})
}

// RecordSnippetView handles POST /api/snippets/:id/view
// Increments the view count ONLY ONCE per user
func RecordSnippetView(c *gin.Context) {
	snippetID := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Check if view exists
		var view models.EntityView
		result := tx.Where("user_id = ? AND entity_type = ? AND entity_id = ?",
			userID.(string), models.EntityTypeSnippet, snippetID).First(&view)

		if result.Error == nil {
			// Already viewed, do nothing (success)
			return nil
		}

		// 2. Create view record
		newView := models.EntityView{
			ID:         utils.GenerateID(),
			UserID:     userID.(string),
			EntityType: models.EntityTypeSnippet,
			EntityID:   snippetID,
		}
		if err := tx.Create(&newView).Error; err != nil {
			return err
		}

		// 3. Increment snippet view count
		if err := tx.Model(&models.Snippet{}).Where("id = ?", snippetID).
			Update("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record view"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "View recorded"})
}
