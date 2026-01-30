package handlers

import (
	"net/http"
	"strconv"
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
	StdinHistory   string   `json:"stdinHistory"`
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
	StdinHistory   string   `json:"stdinHistory"`
	Status         string   `json:"status"`
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
		// Smart Search: Breakdown query into terms to simulate natural language understanding
		terms := strings.Fields(search)
		if len(terms) > 0 {
			var subQueries []string
			var args []interface{}
			for _, term := range terms {
				searchLike := "%" + term + "%"
				subQueries = append(subQueries, "(title ILIKE ? OR description ILIKE ? OR ? = ANY(tags))")
				args = append(args, searchLike, searchLike, term)
			}
			query = query.Where(strings.Join(subQueries, " OR "), args...)
		}
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

	// P0 FIX: Add Pagination to prevent fetching all snippets (DoS risk)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Fetch limit+1 to determine hasMore
	if result := query.Limit(limit + 1).Offset(offset).Find(&snippets); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch snippets"})
		return
	}

	// Determine if there are more results
	hasMore := len(snippets) > limit
	if hasMore {
		snippets = snippets[:limit] // Trim to requested limit
	}

	// Populate ViewerReaction for authenticated users
	if userID, exists := c.Get("userId"); exists {
		var snippetIDs []string
		for _, s := range snippets {
			snippetIDs = append(snippetIDs, s.ID)
		}
		if len(snippetIDs) > 0 {
			reactionMap := make(map[string]string)
			var reactions []models.SnippetReaction
			database.DB.Select("snippet_id", "reaction").Where("user_id = ? AND snippet_id IN ?", userID, snippetIDs).Find(&reactions)
			for _, r := range reactions {
				reactionMap[r.SnippetID] = r.Reaction
			}

			var follows []models.UserLink
			var authorIDs []string
			for _, s := range snippets {
				authorIDs = append(authorIDs, s.AuthorID)
			}
			database.DB.Where("linker_id = ? AND linked_id IN ?", userID, authorIDs).Find(&follows)
			followMap := make(map[string]bool)
			for _, f := range follows {
				followMap[f.LinkedID] = true
			}

			for i := range snippets {
				if r, ok := reactionMap[snippets[i].ID]; ok {
					snippets[i].ViewerReaction = r
				}
				if followMap[snippets[i].AuthorID] {
					snippets[i].Author.IsFollowing = true
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    snippets,
		"page":    page,
		"limit":   limit,
		"hasMore": hasMore,
		// Legacy field for backward compatibility
		"snippets": snippets,
	})
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
		StdinHistory:   input.StdinHistory,
	}

	// Default visibility
	if snippet.Visibility == "" {
		snippet.Visibility = "public"
	}

	// Handle Status (v1.2: allow public direct post)
	// Handle Status (v1.2: allow public direct post)
	if input.Status != "" {
		snippet.Status = input.Status
		if snippet.Status == "PUBLISHED" {
			snippet.Verified = true
			snippet.LastExecutionStatus = "SUCCESS"
		} else {
			snippet.LastExecutionStatus = ""
		}
	} else {
		snippet.Status = "DRAFT"
		snippet.Verified = false
		snippet.LastExecutionStatus = ""
	}

	if result := database.DB.Create(&snippet); result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "A snippet with this title already exists. Please choose a different title."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Reward XP for creating a snippet (if public)
	if snippet.Visibility == "public" {
		database.DB.Model(&models.User{}).Where("id = ?", userID.(string)).Update("xp", gorm.Expr("xp + ?", 50))
		services.LogActivity(userID.(string), models.ActivityNewSnippet, snippet.ID, "Created a new snippet: "+snippet.Title)
	}

	// Check for Badges
	newBadges, _ := services.CheckBadges(userID.(string))
	NotifyNewBadges(userID.(string), newBadges)

	c.JSON(http.StatusCreated, gin.H{
		"snippet":   snippet,
		"newBadges": newBadges,
	})
}

// ForkSnippet handles POST /snippets/:id/fork
func ForkSnippet(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var original models.Snippet
	if err := database.DB.Preload("Author").First(&original, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	// Create the fork
	fork := models.Snippet{
		ID:             utils.GenerateID(),
		Title:          "Fork of " + original.Title,
		Description:    original.Description,
		Language:       original.Language,
		Code:           original.Code,
		Visibility:     original.Visibility,
		AuthorID:       userID.(string),
		ForkedFromID:   &original.ID,
		Tags:           original.Tags,
		OutputSnapshot: original.OutputSnapshot,
		PreviewType:    original.PreviewType,
		Type:           original.Type,
		Difficulty:     original.Difficulty,
		Runtime:        original.Runtime,
		ReferenceURL:   original.ReferenceURL,
		Status:         "DRAFT", // Always start as draft
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if result := database.DB.Create(&fork); result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			// If a fork with the same title exists, append timestamp
			fork.Title = fork.Title + " (" + time.Now().Format("15:04:05") + ")"
			if err := database.DB.Create(&fork).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
	}

	// Increment copy count of original (forking is a form of copying)
	database.DB.Model(&original).Update("copy_count", gorm.Expr("copy_count + 1"))

	// Log activity
	services.LogActivity(userID.(string), models.ActivityFork, fork.ID, "Forked a snippet: "+original.Title)

	// Notify original author
	if original.AuthorID != userID.(string) {
		notification := models.Notification{
			UserID:    original.AuthorID,
			ActorID:   userID.(string),
			Type:      models.NotificationTypeFork,
			SnippetID: &original.ID,
			Message:   "forked your snippet: " + original.Title,
		}
		// ForkSnippet doesn't have a transaction by default, but CreateNotification uses tx.
		// Let's use database.DB if not in tx, or wrap in tx.
		// For simplicity, let's just use database.DB for now if we can't easily get a tx.
		// Wait, CreateNotification expects *gorm.DB.
		CreateNotification(database.DB, notification)
	}

	c.JSON(http.StatusCreated, gin.H{
		"snippet": fork,
	})
}

// GetSnippet handles GET /snippets/:id
func GetSnippet(c *gin.Context) {
	id := c.Param("id")
	var snippet models.Snippet

	if result := database.DB.Preload("Author").First(&snippet, "id = ?", id); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Populate ViewerReaction for authenticated users
	if userID, exists := c.Get("userId"); exists {
		var reaction models.SnippetReaction
		if err := database.DB.Select("reaction").Where("user_id = ? AND snippet_id = ?", userID, snippet.ID).First(&reaction).Error; err == nil {
			snippet.ViewerReaction = reaction.Reaction
		}

		var followCount int64
		database.DB.Model(&models.UserLink{}).Where("linker_id = ? AND linked_id = ?", userID, snippet.AuthorID).Count(&followCount)
		snippet.Author.IsFollowing = followCount > 0
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

	if input.Status != "" {
		snippet.Status = input.Status
		if snippet.Status == "PUBLISHED" {
			snippet.Verified = true
			snippet.LastExecutionStatus = "SUCCESS"
		}
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
	if input.StdinHistory != "" {
		snippet.StdinHistory = input.StdinHistory
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

	// Logic to check ownership or Admin override
	// We need to fetch user role. The context "userId" is just ID string.
	// We should fetch User or check claims if available.
	// For now, let's fetch the user to check role.
	var currentUser models.User
	if err := database.DB.Select("id", "role").First(&currentUser, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if snippet.AuthorID != userID.(string) && currentUser.Role != models.RoleAdmin && currentUser.Role != "STAFF" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own snippets"})
		return
	}

	// Use Transaction for Safe Deletion of cascading dependencies
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Delete Notifications referencing this snippet
		// Fixes: fk_notifications_snippet violation
		if err := tx.Where("snippet_id = ?", snippet.ID).Delete(&models.Notification{}).Error; err != nil {
			return err
		}

		// 2. Delete Comments
		if err := tx.Where("snippet_id = ?", snippet.ID).Delete(&models.Comment{}).Error; err != nil {
			return err
		}

		// 3. Delete Reactions
		if err := tx.Where("snippet_id = ?", snippet.ID).Delete(&models.SnippetReaction{}).Error; err != nil {
			return err
		}

		// 4. Delete the Snippet
		if err := tx.Delete(&snippet).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete snippet: " + err.Error()})
		return
	}

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

	// P0 FIX: Enforce code size limits to prevent Piston abuse
	if len(snippet.Code) > MaxCodeSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Snippet code too large to execute",
			"limit": "64KB maximum",
		})
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

	// Identify Viewer for Blocking Logic
	var viewerID string
	if id, exists := c.Get("userId"); exists {
		viewerID = id.(string)
	} else {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			if claims, err := utils.ValidateToken(authHeader[7:]); err == nil {
				viewerID = claims.UserID
			}
		}
	}

	var snippets []models.Snippet
	query := database.DB.Model(&models.Snippet{}).
		Preload("Author").
		Where("status = ?", "PUBLISHED")

	// Apply Blocking Filter
	if viewerID != "" {
		var blocks []models.UserBlock
		database.DB.Where("blocker_id = ? OR blocked_id = ?", viewerID, viewerID).Find(&blocks)

		var excludedIDs []string
		for _, b := range blocks {
			if b.BlockerID == viewerID {
				excludedIDs = append(excludedIDs, b.BlockedID)
			} else {
				excludedIDs = append(excludedIDs, b.BlockerID)
			}
		}

		if len(excludedIDs) > 0 {
			query = query.Where("\"authorId\" NOT IN ?", excludedIDs)
		}
	}

	switch bucket {
	case "trending":
		// Score = copyCount * 2 + viewsCount - (hours_since_post * 0.05)
		query = query.
			Where("\"createdAt\" > NOW() - INTERVAL '30 days'").
			Order("(copy_count * 2 + views_count - EXTRACT(EPOCH FROM (NOW() - \"createdAt\"))/3600 * 0.05) DESC")
	case "personal":
		if viewerID != "" {
			var follows []models.UserLink
			database.DB.Where("linker_id = ?", viewerID).Find(&follows)
			var followingIDs []string
			for _, f := range follows {
				followingIDs = append(followingIDs, f.LinkedID)
			}
			if len(followingIDs) > 0 {
				query = query.Where("\"authorId\" IN ?", followingIDs)
			} else {
				query = query.Where("1 = 0")
			}
		}
		query = query.Order("\"createdAt\" DESC")
	case "new":
		query = query.Order("\"createdAt\" DESC")
	case "editor":
		query = query.Where("is_featured = true").Order("\"createdAt\" DESC")
	default:
		query = query.Order("\"createdAt\" DESC")
	}

	// P0 FIX: Add Pagination to Feed (replace hardcoded limit)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50 // Stricter limit for feed
	}
	offset := (page - 1) * limit

	// Fetch limit+1 to determine hasMore
	if err := query.Limit(limit + 1).Offset(offset).Find(&snippets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	// Determine if there are more results
	hasMore := len(snippets) > limit
	if hasMore {
		snippets = snippets[:limit] // Trim to requested limit
	}

	// Populate ViewerReaction for authenticated users
	if viewerID != "" {
		var snippetIDs []string
		for _, s := range snippets {
			snippetIDs = append(snippetIDs, s.ID)
		}

		if len(snippetIDs) > 0 {
			var reactions []models.SnippetReaction
			database.DB.Select("snippet_id", "reaction").Where("user_id = ? AND snippet_id IN ?", viewerID, snippetIDs).Find(&reactions)

			reactionMap := make(map[string]string)
			for _, r := range reactions {
				reactionMap[r.SnippetID] = r.Reaction
			}

			var follows []models.UserLink
			var authorIDs []string
			for _, s := range snippets {
				authorIDs = append(authorIDs, s.AuthorID)
			}
			database.DB.Where("linker_id = ? AND linked_id IN ?", viewerID, authorIDs).Find(&follows)
			followMap := make(map[string]bool)
			for _, f := range follows {
				followMap[f.LinkedID] = true
			}

			// Map back to result
			for i := range snippets {
				if r, ok := reactionMap[snippets[i].ID]; ok {
					snippets[i].ViewerReaction = r
				}
				if followMap[snippets[i].AuthorID] {
					snippets[i].Author.IsFollowing = true
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    snippets,
		"page":    page,
		"limit":   limit,
		"hasMore": hasMore,
		"bucket":  bucket,
		// Legacy field for backward compatibility
		"snippets": snippets,
	})
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

// GetSimilarSnippets handles GET /snippets/:id/similar
func GetSimilarSnippets(c *gin.Context) {
	id := c.Param("id")
	var snippet models.Snippet
	if err := database.DB.First(&snippet, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snippet not found"})
		return
	}

	var similar []models.Snippet
	// Semantic Similarity: Same language or overlapping tags
	query := database.DB.Model(&models.Snippet{}).Preload("Author").
		Where("id <> ? AND (language = ? OR tags && ?)", id, snippet.Language, snippet.Tags).
		Order("views_count DESC").
		Limit(5)

	if err := query.Find(&similar).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch similar snippets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snippets": similar})
}
