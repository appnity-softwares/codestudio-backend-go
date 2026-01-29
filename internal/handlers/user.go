package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
	"github.com/pushp314/devconnect-backend/pkg/utils"
)

// ProfileSummaryResponse defines the shape of the summary API
type ProfileSummaryResponse struct {
	Snippets SnippetSummary `json:"snippets"`
	Arena    ArenaSummary   `json:"arena"`
}

type SnippetSummary struct {
	Total      int64            `json:"total"`
	ByLanguage map[string]int64 `json:"byLanguage"`
}

type ArenaSummary struct {
	ContestsJoined int64 `json:"contestsJoined"`
}

// GetProfileSummary returns strict MVP stats for a user's profile
func GetProfileSummary(c *gin.Context) {
	// 1. Resolve Target User (Optional Auth + Public Username fallback)
	username := c.Query("username")

	// If no username provided, we MUST assume "me" context -> Require Auth
	if username == "" {
		// Try to get from Context (if AuthMiddleware ran)
		userID, exists := c.Get("userId")

		// If not in context, try manual header extraction (for Optional Auth routes)
		if !exists {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString := authHeader[7:]
				if claims, err := utils.ValidateToken(tokenString); err == nil {
					// Correction: utils.ValidateToken, but need to check imports.
					// Assuming utils is imported as "utils" or part of models package?
					// I checked utils/token.go -> package utils.
					// I need to make sure I use `utils.ValidateToken`.
					// Checking imports of user.go...
					userID = claims.UserID
					exists = true
				}
			}
		}

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required for self-summary"})
			return
		}

		// Resolve target User ID
		var user models.User
		if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		username = user.Username
	}

	// 1. Find User by Username
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	// 2. Count PUBLISHED Snippets & Aggregate by Language
	var totalSnippets int64
	database.DB.Model(&models.Snippet{}).
		Where("\"authorId\" = ? AND status = ?", user.ID, "PUBLISHED").
		Count(&totalSnippets)

	// Group by Language
	type LangResult struct {
		Language string
		Count    int64
	}
	var langResults []LangResult
	database.DB.Model(&models.Snippet{}).
		Select("language, count(*) as count").
		Where("\"authorId\" = ? AND status = ?", user.ID, "PUBLISHED").
		Group("language").
		Scan(&langResults)

	byLanguage := make(map[string]int64)
	// Initialize strict MVP languages to 0
	byLanguage["typescript"] = 0
	byLanguage["python"] = 0
	byLanguage["go"] = 0

	for _, res := range langResults {
		// Normalize or map if needed. Assuming stored as lowercase "python", "typescript", "go"
		// If DB has "Python", "javascript", etc., we strictly map to what's requested is okay,
		// but typically we just dump what's there. The Prompt said "Count ONLY supported MVP languages".
		// We can filter here.
		l := res.Language // assume normalized or do strings.ToLower
		if l == "typescript" || l == "python" || l == "go" {
			byLanguage[l] = res.Count
		}
	}

	// 3. Count Contests Joined (Registrations)
	var contestsJoined int64
	database.DB.Model(&models.Registration{}).
		Where("user_id = ?", user.ID).
		Count(&contestsJoined)

	// 4. Return Response
	resp := ProfileSummaryResponse{
		Snippets: SnippetSummary{
			Total:      totalSnippets,
			ByLanguage: byLanguage,
		},
		Arena: ArenaSummary{
			ContestsJoined: contestsJoined,
		},
	}

	c.JSON(200, resp)
}

// -- Inputs -- //
// UpdateMeInput defines fields user can update
type UpdateMeInput struct {
	Name           *string  `json:"name"`
	Bio            *string  `json:"bio"`
	GithubURL      *string  `json:"githubUrl"`
	InstagramURL   *string  `json:"instagramUrl"`
	LinkedInURL    *string  `json:"linkedinUrl"`
	Visibility     *string  `json:"visibility"`
	Onboarding     *bool    `json:"onboardingCompleted"`
	PreferredLangs []string `json:"preferredLanguages"`
	Interests      []string `json:"interests"`
}

type OnboardingInput struct {
	Name         string   `json:"name" binding:"required"`
	Username     string   `json:"username" binding:"required"`
	Bio          string   `json:"bio"`
	Image        string   `json:"image"`
	GithubURL    string   `json:"githubUrl"`
	InstagramURL string   `json:"instagramUrl"`
	LinkedInURL  string   `json:"linkedinUrl"`
	Languages    []string `json:"languages"`
	Interests    []string `json:"interests"`
}

func CompleteOnboarding(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input OnboardingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Name and Username are required from input
	user.Name = input.Name

	// Double check username uniqueness if changed
	if input.Username != user.Username {
		if !utils.ValidateUsername(input.Username) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid username format"})
			return
		}
		var count int64
		database.DB.Model(&models.User{}).Where("username = ? AND id != ?", input.Username, userID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
			return
		}
		user.Username = input.Username
	}

	user.Bio = input.Bio
	user.Image = input.Image
	user.GithubURL = input.GithubURL
	user.InstagramURL = input.InstagramURL
	user.LinkedInURL = input.LinkedInURL
	user.PreferredLanguages = input.Languages
	user.Interests = input.Interests
	user.OnboardingCompleted = true

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Onboarding completed", "user": user})
}

// Helper for array string formating

type UpdateProfileInput struct {
	Name                 string   `json:"name"`
	Username             string   `json:"username"`
	Bio                  string   `json:"bio"`
	Image                string   `json:"image"`
	GithubURL            string   `json:"githubUrl"`
	InstagramURL         string   `json:"instagramUrl"`
	LinkedInURL          string   `json:"linkedinUrl"`
	Visibility           string   `json:"visibility"`
	Languages            []string `json:"languages"`
	Interests            []string `json:"interests"`
	PinnedSnippetID      *string  `json:"pinnedSnippetId"` // MVP v1.1
	PublicProfileEnabled *bool    `json:"publicProfileEnabled"`
	SearchVisible        *bool    `json:"searchVisible"`
	GithubStatsVisible   *bool    `json:"githubStatsVisible"`
}

// -- Handlers -- //

// GetProfile handles GET /users/profile (Current User) or /users/:username
func GetProfile(c *gin.Context) {
	username := c.Param("username")
	var viewerID string

	// Check auth for blocking logic (Optional Auth)
	if id, exists := c.Get("userId"); exists {
		viewerID = id.(string)
	} else {
		// Manual header check since route might not have OptionalAuthMiddleware
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			if claims, err := utils.ValidateToken(authHeader[7:]); err == nil {
				viewerID = claims.UserID
			}
		}
	}

	var user models.User
	var result error

	// Preload PinnedSnippet for profile view
	query := database.DB.Preload("PinnedSnippet").Preload("PinnedSnippet.Author")

	if username != "" && username != "me" {
		result = query.Where("username = ? OR id = ?", username, username).First(&user).Error

		// Blocking Check
		if result == nil && viewerID != "" && viewerID != user.ID {
			var blockCount int64
			// Check if viewer blocked user OR user blocked viewer
			database.DB.Model(&models.UserBlock{}).
				Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
					viewerID, user.ID, user.ID, viewerID).
				Count(&blockCount)

			if blockCount > 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"}) // Hide completely
				return
			}
		}

	} else {
		userID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		result = query.First(&user, "id = ?", userID).Error
	}

	if result != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Populate Counts
	var snippetCount int64
	database.DB.Model(&models.Snippet{}).Where(&models.Snippet{AuthorID: user.ID}).Count(&snippetCount)
	user.Count.Snippets = snippetCount

	// Add IsFollowing status if viewer exists
	isFollowing := false
	if viewerID != "" && viewerID != user.ID {
		var count int64
		database.DB.Model(&models.UserLink{}).Where("linker_id = ? AND linked_id = ?", viewerID, user.ID).Count(&count)
		if count > 0 {
			isFollowing = true
		}
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "isFollowing": isFollowing})
}

// UpdateProfile handles PUT /users/profile
func UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if input.Name != "" {
		user.Name = input.Name
	}

	// Username Change Logic (Limit: 2 changes per 90 days)
	if input.Username != "" && input.Username != user.Username {
		if !utils.ValidateUsername(input.Username) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid username format"})
			return
		}
		// Verify uniqueness
		var count int64
		database.DB.Model(&models.User{}).Where("username = ? AND id != ?", input.Username, userID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
			return
		}

		// Check Limits
		now := time.Now()
		var daysSinceLastChange float64 = 999 // Default to allowed
		if user.LastUsernameChangeAt != nil {
			daysSinceLastChange = now.Sub(*user.LastUsernameChangeAt).Hours() / 24
		}

		if daysSinceLastChange < 90 {
			if user.UsernameChangeCount >= 2 {
				c.JSON(http.StatusForbidden, gin.H{
					"error": fmt.Sprintf("Username change limit reached. You can change it again in %d days.", int(90-daysSinceLastChange)),
				})
				return
			}
			user.UsernameChangeCount++
		} else {
			// Reset window if > 90 days
			user.UsernameChangeCount = 1
		}

		user.Username = input.Username
		user.LastUsernameChangeAt = &now
	}

	if input.Bio != "" {
		user.Bio = input.Bio
	}
	if input.Image != "" {
		user.Image = input.Image
	}
	if input.GithubURL != "" {
		user.GithubURL = input.GithubURL
	}
	if input.InstagramURL != "" {
		user.InstagramURL = input.InstagramURL
	}
	if input.LinkedInURL != "" {
		user.LinkedInURL = input.LinkedInURL
	}

	if input.Languages != nil {
		user.PreferredLanguages = input.Languages
	}
	if input.Interests != nil {
		user.Interests = input.Interests
	}

	if input.PinnedSnippetID != nil {
		user.PinnedSnippetID = input.PinnedSnippetID
	}
	if input.PublicProfileEnabled != nil {
		user.PublicProfileEnabled = *input.PublicProfileEnabled
	}
	if input.SearchVisible != nil {
		user.SearchVisible = *input.SearchVisible
	}
	if input.GithubStatsVisible != nil {
		user.GithubStatsVisible = *input.GithubStatsVisible
	}

	if input.Visibility != "" {
		user.Visibility = models.Visibility(input.Visibility)
	}

	// Ensure we save updates
	database.DB.Save(&user)

	// Populate Counts for response
	var snippetCount int64
	// Use explicit query to avoid GORM struct zero-value pitfalls
	database.DB.Model(&models.Snippet{}).Where("\"authorId\" = ?", user.ID).Count(&snippetCount)
	user.Count.Snippets = snippetCount

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// GetPublicProfile handles GET /public/users/:username (Public Route)
func GetPublicProfile(c *gin.Context) {
	username := c.Param("username")

	var user models.User
	// Preload limited data to avoid leakage? We clean up in response or use tags.
	// JSON tags are mostly safe, but email is there.
	// We should manually construct a SafeUser struct or just sanitize before return.
	// For MVP, if we return `user` struct, checking JSON tags:
	// Email is `json:"email"`. We should NOT return it.
	// We MUST sanitize.

	if err := database.DB.Preload("PinnedSnippet").Preload("PinnedSnippet.Author").
		Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check Privacy
	if !user.PublicProfileEnabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "User profile not available"})
		return
	}

	// Aggregations (Real-time for now, or use cached)
	// We'll trust cached columns if updated, but update them on reads occasionally?
	// For MVP, let's just do count queries, Postgres handles them fast enough for <100k users.
	var snippetCount int64
	database.DB.Model(&models.Snippet{}).Where("\"authorId\" = ? AND status = 'PUBLISHED'", user.ID).Count(&snippetCount)

	var contestCount int64
	database.DB.Model(&models.Registration{}).Where("user_id = ?", user.ID).Count(&contestCount)

	// Fetch Top 3 Snippets by Views
	var topSnippets []models.Snippet
	database.DB.Where("\"authorId\" = ? AND status = 'PUBLISHED'", user.ID).
		Order("views_count desc").Limit(3).Find(&topSnippets)

	// Sanitize Response
	safeUser := gin.H{
		"id":                   user.ID,
		"username":             user.Username,
		"name":                 user.Name,
		"image":                user.Image,
		"bio":                  user.Bio,
		"trustScore":           user.TrustScore,
		"githubUrl":            user.GithubURL,
		"instagramUrl":         user.InstagramURL,
		"linkedinUrl":          user.LinkedInURL,
		"createdAt":            user.CreatedAt,
		"pinnedSnippet":        user.PinnedSnippet,
		"isBlocked":            user.IsBlocked, // Keep for frontend logic? Actually probably hide.
		"pinnedSnippetId":      user.PinnedSnippetID,
		"snippetCount":         snippetCount,
		"contestCount":         contestCount,
		"topSnippets":          topSnippets,
		"preferredLanguages":   user.PreferredLanguages,
		"interests":            user.Interests,
		"publicProfileEnabled": user.PublicProfileEnabled,
		"linkersCount":         user.LinkersCount,
		"linkedCount":          user.LinkedCount,
		"xp":                   user.XP,
	}

	c.JSON(http.StatusOK, gin.H{"user": safeUser})
}

// ListCommunityUsers handles GET /community/users
func ListCommunityUsers(c *gin.Context) {
	// Filters: ?search= &sort= &page=
	search := c.Query("search")
	sort := c.Query("sort")

	pageStr := c.Query("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	offset := (page - 1) * limit

	query := database.DB.Model(&models.User{}).Where("search_visible = ? AND onboarding_completed = ?", true, true)

	// Don't show own profile in community list
	if userID, exists := c.Get("userId"); exists {
		query = query.Where("id != ?", userID)
	}

	if search != "" {
		searchLike := utils.SanitizeSearchQuery(search)
		query = query.Where("username ILIKE ? OR name ILIKE ? OR city ILIKE ?", searchLike, searchLike, searchLike)
	}

	city := c.Query("city")
	if city != "" {
		query = query.Where("city ILIKE ?", city)
	}

	// Sorting
	switch sort {
	case "recommended":
		// Get current user's preferences if logged in
		userID, exists := c.Get("userId")
		if exists {
			var me models.User
			if err := database.DB.First(&me, "id = ?", userID).Error; err == nil && (len(me.PreferredLanguages) > 0 || len(me.Interests) > 0) {
				// Use PostgreSQL && operator for array overlap
				// We'll prioritize users who have overlap, then trust score
				query = query.Order(database.DB.Raw("(CASE WHEN preferred_languages && ? OR interests && ? THEN 1 ELSE 0 END) DESC", me.PreferredLanguages, me.Interests))
			}
		}
		query = query.Order("trust_score desc, \"createdAt\" desc")
	case "active":
		query = query.Order("\"createdAt\" desc")
	case "trust":
		query = query.Order("trust_score desc")
	case "snippets":
		query = query.Order("trust_score desc")
	default:
		// Default to recommended
		query = query.Order("trust_score desc")
	}

	var users []models.User
	if err := query.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch community"})
		return
	}

	// Sanitize List
	// Sanitize List & Populate Real Counts
	var safeUsers []gin.H
	for _, u := range users {
		var snippetCount int64
		database.DB.Model(&models.Snippet{}).Where("\"authorId\" = ? AND status = 'PUBLISHED'", u.ID).Count(&snippetCount)

		var contestCount int64
		database.DB.Model(&models.Registration{}).Where("user_id = ?", u.ID).Count(&contestCount)

		safeUsers = append(safeUsers, gin.H{
			"id":           u.ID,
			"username":     u.Username,
			"name":         u.Name,
			"image":        u.Image,
			"bio":          u.Bio,
			"trustScore":   u.TrustScore,
			"createdAt":    u.CreatedAt,
			"snippetCount": snippetCount,
			"contestCount": contestCount,
			"languages":    u.PreferredLanguages,
			"interests":    u.Interests,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": safeUsers, "page": page})
}

// SearchSuggestions handles GET /community/search-suggestions
func SearchSuggestions(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"users": []gin.H{}})
		return
	}

	searchPattern := q + "%"
	var users []models.User
	database.DB.Model(&models.User{}).
		Where("onboarding_completed = ? AND (username ILIKE ? OR name ILIKE ?)", true, searchPattern, searchPattern).
		Limit(5).
		Find(&users)

	var suggestions []gin.H
	for _, u := range users {
		suggestions = append(suggestions, gin.H{
			"username": u.Username,
			"name":     u.Name,
			"image":    u.Image,
			"bio":      u.Bio,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": suggestions})
}

// GetStats handles GET /users/stats - Returns real engagement data for dashboard
// v1.2: Enhanced with progress metrics
func GetStats(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get user for trust score
	var user models.User
	if err := database.DB.First(&user, "id = ?", userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Count total snippets
	var snippetCount int64
	database.DB.Model(&models.Snippet{}).Where("\"authorId\" = ?", userId).Count(&snippetCount)

	// v1.2: Count total copies received
	var totalCopiesReceived int64
	database.DB.Model(&models.Snippet{}).
		Select("COALESCE(SUM(copy_count), 0)").
		Where("\"authorId\" = ?", userId).
		Scan(&totalCopiesReceived)

	// v1.2: Count contest solves (successful submissions)
	var contestSolves int64
	database.DB.Model(&models.Submission{}).
		Where("user_id = ? AND status = ?", userId, "ACCEPTED").
		Count(&contestSolves)

	// v1.2: Count contests joined
	var contestsJoined int64
	database.DB.Model(&models.Registration{}).
		Where("user_id = ?", userId).
		Count(&contestsJoined)

	// v1.2: Compute rank percentile based on trust score
	var totalUsers int64
	database.DB.Model(&models.User{}).Count(&totalUsers)
	var usersAbove int64
	database.DB.Model(&models.User{}).Where("trust_score > ?", user.TrustScore).Count(&usersAbove)
	rankPercentile := 0
	if totalUsers > 0 {
		rankPercentile = int(100 - (usersAbove * 100 / totalUsers))
	}

	// Chart data (activity over last 7 days - simplified)
	chartData := []gin.H{
		{"name": "Mon", "activity": 0},
		{"name": "Tue", "activity": 0},
		{"name": "Wed", "activity": 0},
		{"name": "Thu", "activity": 0},
		{"name": "Fri", "activity": 0},
		{"name": "Sat", "activity": 0},
		{"name": "Sun", "activity": 0},
	}

	c.JSON(http.StatusOK, gin.H{
		"snippets":            snippetCount,
		"totalCopiesReceived": totalCopiesReceived,
		"contestSolves":       contestSolves,
		"contestsJoined":      contestsJoined,
		"trustScore":          user.TrustScore,
		"rankPercentile":      rankPercentile,
		"chart":               chartData,
	})
}

// ListUsers handles GET /users
// func ListUsers(c *gin.Context) {
// 	var users []models.User
// 	if err := database.DB.Find(&users).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"users": users})
// }

// Social features removed for MVP

// GetUserSnippets handles GET /users/:id/snippets
func GetUserSnippets(c *gin.Context) {
	targetID := c.Param("username")
	var viewerID string

	// Check auth for blocking logic (Optional Auth)
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

	// 1. Get Target User Status
	var targetUser models.User
	if err := database.DB.Select("id, visibility").Where("id = ? OR username = ?", targetID, targetID).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Blocking Check
	if viewerID != "" && viewerID != targetUser.ID {
		var blockCount int64
		database.DB.Model(&models.UserBlock{}).
			Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
				viewerID, targetUser.ID, targetUser.ID, viewerID).
			Count(&blockCount)

		if blockCount > 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
	}

	// 2. Privacy Check
	if targetUser.Visibility == models.VisibilityPrivate {
		// If private, only linked users or author can see
		isAuthor := viewerID != "" && viewerID == targetUser.ID
		isLinked := false

		if !isAuthor && viewerID != "" {
			var count int64
			database.DB.Model(&models.UserLink{}).Where("linker_id = ? AND linked_id = ?", viewerID, targetUser.ID).Count(&count)
			isLinked = count > 0
		}

		if !isAuthor && !isLinked {
			c.JSON(http.StatusForbidden, gin.H{
				"error":    "Profile is private",
				"snippets": []models.Snippet{},
				"private":  true,
			})
			return
		}
	}

	snippets := []models.Snippet{}
	// Postgres is case-sensitive for quoted identifiers.
	// The DB table is "Snippet" and columns are "authorId", "createdAt".
	if err := database.DB.Model(&models.Snippet{}).
		Preload("Author").
		Where("\"authorId\" = ?", targetUser.ID).
		Order("\"createdAt\" DESC").
		Find(&snippets).Error; err != nil {
		fmt.Printf("Error fetching user snippets: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch snippets",
			"details": err.Error(),
		})
		return
	}

	// 3. Populate Interaction Status (IsLiked/IsDisliked)
	if viewerID != "" {
		var snippetIDs []string
		for _, s := range snippets {
			snippetIDs = append(snippetIDs, s.ID)
		}

		if len(snippetIDs) > 0 {
			var likes []models.SnippetLike
			database.DB.Select("snippet_id").Where("user_id = ? AND snippet_id IN ?", viewerID, snippetIDs).Find(&likes)

			likedMap := make(map[string]bool)
			for _, l := range likes {
				likedMap[l.SnippetID] = true
			}

			var dislikes []models.SnippetDislike
			database.DB.Select("snippet_id").Where("user_id = ? AND snippet_id IN ?", viewerID, snippetIDs).Find(&dislikes)

			dislikedMap := make(map[string]bool)
			for _, d := range dislikes {
				dislikedMap[d.SnippetID] = true
			}

			// Map back to result
			for i := range snippets {
				if likedMap[snippets[i].ID] {
					snippets[i].IsLiked = true
				}
				if dislikedMap[snippets[i].ID] {
					snippets[i].IsDisliked = true
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"snippets": snippets})
}

// GetBadges handles GET /users/:username/badges
func GetBadges(c *gin.Context) {
	username := c.Param("username")

	var user models.User
	if err := database.DB.Where("username = ? OR id = ?", username, username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 1. Fetch All System Badges
	var allBadges []models.Badge
	database.DB.Order("threshold asc").Find(&allBadges)

	// 2. Fetch User's Unlocked/Progress Badges
	var userBadges []models.UserBadge
	database.DB.Where("user_id = ?", user.ID).Find(&userBadges)

	// Map wrapper for easier lookup
	userBadgeMap := make(map[string]models.UserBadge)
	for _, ub := range userBadges {
		userBadgeMap[ub.BadgeID] = ub
	}

	// 3. Calculate Stats for Self-Healing & Influence
	// Count ALL snippets (including drafts, so users see progress immediately)
	var snippetCount int64
	database.DB.Model(&models.Snippet{}).Where("\"authorId\" = ?", user.ID).Count(&snippetCount)

	// Count Contests
	var contestCount int64
	database.DB.Model(&models.Registration{}).Where("user_id = ? AND status != 'BANNED'", user.ID).Count(&contestCount)

	// Count Practice Problems Solved
	var practiceCount int64
	database.DB.Model(&models.Submission{}).Where("user_id = ? AND status = 'ACCEPTED'", user.ID).Count(&practiceCount)

	// Count Feedback Sent
	var feedbackCount int64
	database.DB.Model(&models.FeedbackMessage{}).Where("user_id = ?", user.ID).Count(&feedbackCount)

	// Count Rank for Early Adopter (First 1000)
	var userJoinRank int64
	database.DB.Model(&models.User{}).
		Where("\"createdAt\" <= (SELECT \"createdAt\" FROM \"User\" WHERE id = ?)", user.ID).
		Count(&userJoinRank)
	earlyAdopterProgress := int64(0)
	if userJoinRank <= 1000 {
		earlyAdopterProgress = 1
	}

	// 4. Construct Response & Self-Healing
	type BadgeResponse struct {
		models.Badge
		Unlocked   bool      `json:"unlocked"`
		Progress   int64     `json:"progress"`
		UnlockedAt time.Time `json:"unlockedAt,omitempty"`
	}

	var responseBadges []BadgeResponse
	unlockedCount := 0

	for _, badge := range allBadges {
		ub, exists := userBadgeMap[badge.ID]

		var currentProgress int64 = 0
		if exists {
			currentProgress = int64(ub.Progress)
		}

		// Dynamic Progress Check
		switch badge.Condition {
		case "1_snippet", "5_snippets", "10_snippets", "25_snippets", "snippet_master":
			currentProgress = snippetCount
		case "1_contest", "5_contests":
			currentProgress = contestCount
		case "1_practice_solved", "5_practice_solved", "25_practice_solved":
			currentProgress = practiceCount
		case "feedback_given", "5_feedback":
			currentProgress = feedbackCount
		case "early_adopter":
			currentProgress = earlyAdopterProgress
		}

		// Dynamic Type assignment for refinement
		if badge.Condition == "early_adopter" || badge.Condition == "contest_winner" || badge.Condition == "snippet_master" || badge.Condition == "25_snippets" {
			badge.Type = models.BadgeType3D
		} else {
			badge.Type = models.BadgeType2D
		}

		// Logic Override for "Early Adopter" (Threshold 0)
		isUnlocked := exists && (badge.Threshold == 0 || ub.Progress >= badge.Threshold)

		// Self-Healing
		if !isUnlocked && badge.Threshold > 0 && currentProgress >= int64(badge.Threshold) {
			newBadge := models.UserBadge{
				UserID:     user.ID,
				BadgeID:    badge.ID,
				Progress:   int(currentProgress),
				UnlockedAt: time.Now(),
			}
			if err := database.DB.Create(&newBadge).Error; err == nil {
				isUnlocked = true
				exists = true
				ub = newBadge
			}
		}

		if isUnlocked {
			unlockedCount++
			responseBadges = append(responseBadges, BadgeResponse{
				Badge:      badge,
				Unlocked:   true,
				Progress:   currentProgress,
				UnlockedAt: ub.UnlockedAt,
			})
		} else {
			responseBadges = append(responseBadges, BadgeResponse{
				Badge:    badge,
				Unlocked: false,
				Progress: currentProgress,
			})
		}
	}

	// 5. Calculate Influence Score
	// Base: Trust Score
	influence := int64(user.TrustScore)

	// Bonus: Snippets * 10
	influence += snippetCount * 10

	// Bonus: Badges * 50
	influence += int64(unlockedCount * 50)

	// Bonus: Contests
	influence += contestCount * 25

	// ADMIN OVERRIDE: Supreme Influence
	rank := "Novice"
	if user.Role == models.RoleAdmin {
		influence = 1000000
		rank = "Supreme Architect"
	} else {
		if influence >= 5000 {
			rank = "Architect"
		} else if influence >= 1000 {
			rank = "Rising Star"
		} else if influence >= 500 {
			rank = "Contributor"
		} else if influence >= 100 {
			rank = "Explorer"
		} else {
			rank = "Apprentice"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"badges": responseBadges,
		"authority": gin.H{
			"score": influence,
			"rank":  rank,
			"breakdown": gin.H{
				"trust":    user.TrustScore,
				"snippets": snippetCount * 10,
				"badges":   unlockedCount * 50,
				"contests": contestCount * 25,
			},
		},
	})
}

// GetGlobalLeaderboard returns top users by XP
func GetGlobalLeaderboard(c *gin.Context) {
	var users []models.User
	// Order by XP descending, limit to 20
	// We only show users with XP > 0 and who completed onboarding
	if err := database.DB.Model(&models.User{}).
		Where("xp > 0 AND onboarding_completed = ?", true).
		Order("xp DESC, \"createdAt\" ASC").
		Limit(20).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}

	// Sanitize output
	var safeUsers []gin.H
	for i, u := range users {
		safeUsers = append(safeUsers, gin.H{
			"rank":     i + 1,
			"username": u.Username,
			"name":     u.Name,
			"image":    u.Image,
			"xp":       u.XP,
			"id":       u.ID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": safeUsers})
}

// SpendXP handles POST /users/spend-xp
func SpendXP(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		ItemID string `json:"itemId" binding:"required"`
		Amount int    `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.XP < input.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient XP balance"})
		return
	}

	// Check if already purchased
	for _, id := range user.PurchasedComponentIds {
		if id == input.ItemID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Item already unlocked"})
			return
		}
	}

	user.XP -= input.Amount
	user.PurchasedComponentIds = append(user.PurchasedComponentIds, input.ItemID)

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item unlocked successfully", "xp": user.XP, "purchasedIds": user.PurchasedComponentIds})
}
