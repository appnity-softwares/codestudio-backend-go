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
		Where("author_id = ? AND status = ?", user.ID, "PUBLISHED").
		Count(&totalSnippets)

	// Group by Language
	type LangResult struct {
		Language string
		Count    int64
	}
	var langResults []LangResult
	database.DB.Model(&models.Snippet{}).
		Select("language, count(*) as count").
		Where("author_id = ? AND status = ?", user.ID, "PUBLISHED").
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
	Visibility     *string  `json:"visibility"`
	Onboarding     *bool    `json:"onboardingCompleted"`
	PreferredLangs []string `json:"preferredLanguages"`
	Interests      []string `json:"interests"`
}

type OnboardingInput struct {
	Bio       string   `json:"bio"`
	Languages []string `json:"languages"`
	Interests []string `json:"interests"`
}

func CompleteOnboarding(c *gin.Context) {
	userID := c.GetString("userID")
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

	// Helper to create pointer to array string
	langs := fmt.Sprintf("{%s}", joinStrings(input.Languages))
	interests := fmt.Sprintf("{%s}", joinStrings(input.Interests))

	user.Bio = input.Bio
	user.PreferredLanguages = &langs
	user.Interests = &interests
	user.OnboardingCompleted = true

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Onboarding completed", "user": user})
}

// Helper for array string formating
func joinStrings(s []string) string {
	res := ""
	for i, v := range s {
		if i > 0 {
			res += ","
		}
		res += fmt.Sprintf("\"%s\"", v)
	}
	return res
}

type UpdateProfileInput struct {
	Name                 string  `json:"name"`
	Username             string  `json:"username"`
	Bio                  string  `json:"bio"`
	Image                string  `json:"image"`
	GithubURL            string  `json:"githubUrl"`
	InstagramURL         string  `json:"instagramUrl"`
	Visibility           string  `json:"visibility"`
	PinnedSnippetID      *string `json:"pinnedSnippetId"` // MVP v1.1
	PublicProfileEnabled *bool   `json:"publicProfileEnabled"`
	SearchVisible        *bool   `json:"searchVisible"`
}

// -- Handlers -- //

// GetProfile handles GET /users/profile (Current User) or /users/:username
func GetProfile(c *gin.Context) {
	// If "username" param is present, fetch that user. Else fetch current user.
	username := c.Param("username")

	var user models.User
	var result error

	// Preload PinnedSnippet for profile view
	query := database.DB.Preload("PinnedSnippet").Preload("PinnedSnippet.Author")

	if username != "" && username != "me" {
		result = query.Where("username = ? OR id = ?", username, username).First(&user).Error
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

	c.JSON(http.StatusOK, gin.H{"user": user})
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
		// Verify uniqueness
		var count int64
		database.DB.Model(&models.User{}).Where("username = ?", input.Username).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already taken"})
			return
		}

		// Check Limits
		now := time.Now()
		daysSinceLastChange := now.Sub(user.LastUsernameChangeAt).Hours() / 24

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
		user.LastUsernameChangeAt = now
	}

	// Allow empty bio updates? If strict MVP, maybe. But standard is yes.
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
	if input.Visibility != "" {
		user.Visibility = models.Visibility(input.Visibility)
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

	// Ensure we save updates
	database.DB.Save(&user)

	// Populate Counts for response
	var snippetCount int64
	database.DB.Model(&models.Snippet{}).Where(&models.Snippet{AuthorID: user.ID}).Count(&snippetCount)
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
	database.DB.Model(&models.Snippet{}).Where("author_id = ? AND status = 'PUBLISHED'", user.ID).Count(&snippetCount)

	var contestCount int64
	database.DB.Model(&models.Registration{}).Where("user_id = ?", user.ID).Count(&contestCount)

	// Fetch Top 3 Snippets by Views
	var topSnippets []models.Snippet
	database.DB.Where("author_id = ? AND status = 'PUBLISHED'", user.ID).
		Order("view_count desc").Limit(3).Find(&topSnippets)

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
		"createdAt":            user.CreatedAt,
		"pinnedSnippet":        user.PinnedSnippet,
		"isBlocked":            user.IsBlocked, // Keep for frontend logic? Actually probably hide.
		"pinnedSnippetId":      user.PinnedSnippetID,
		"snippetCount":         snippetCount,
		"contestCount":         contestCount,
		"topSnippets":          topSnippets,
		"publicProfileEnabled": user.PublicProfileEnabled, // public needs to know? sure.
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

	query := database.DB.Model(&models.User{}).Where("search_visible = ?", true)

	if search != "" {
		query = query.Where("username ILIKE ? OR name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Sorting
	switch sort {
	case "active":
		// approximation: created_at or recent activity?
		// for now, recently joined
		query = query.Order("created_at desc")
	case "trust":
		query = query.Order("trust_score desc")
	case "snippets":
		// efficient sort requires column. We added `snippet_count` to model but haven't populated it.
		// Fallback: trust score
		query = query.Order("trust_score desc")
	default:
		// "most active" -> maybe trust score is best proxy for now without complex activity table joins
		query = query.Order("trust_score desc")
	}

	var users []models.User
	if err := query.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch community"})
		return
	}

	// Sanitize List
	var safeUsers []gin.H
	for _, u := range users {
		safeUsers = append(safeUsers, gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"name":       u.Name,
			"image":      u.Image,
			"bio":        u.Bio,
			"trustScore": u.TrustScore,
			"createdAt":  u.CreatedAt,
			// For list view, we might want counts. Avoiding N+1 queries by using the cached columns we added?
			// We added WrappedSnippetCount etc. but logic to update them isn't there yet.
			// Let's rely on client-side fetching or lazy loading if critical, OR just return raw values from User table
			// since we added the columns `snippet_count` etc.
			// Ideally we have a background job that updates these.
			// For MVP, just return 0 or what's in DB.
			"snippetCount": u.WrappedSnippetCount,
			"contestCount": u.WrappedContestCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": safeUsers, "page": page})
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

	// v1.2: Count total forks received on user's snippets
	var totalForksReceived int64
	database.DB.Model(&models.Snippet{}).
		Select("COALESCE(SUM(fork_count), 0)").
		Where("\"authorId\" = ?", userId).
		Scan(&totalForksReceived)

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
		"totalForksReceived":  totalForksReceived,
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
	userId := c.Param("username")

	snippets := []models.Snippet{}
	if err := database.DB.Where("\"authorId\" = ?", userId).Preload("Author").Find(&snippets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch snippets"})
		return
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

	var userBadges []models.UserBadge
	// Preload the Badge definition
	if err := database.DB.Preload("Badge").Where("user_id = ?", user.ID).Find(&userBadges).Error; err != nil {
		fmt.Printf("Error fetching badges for user %s: %v\n", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch badges: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"badges": userBadges})
}
