package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

type GithubStatsData struct {
	PublicRepos   int       `json:"public_repos"`
	Followers     int       `json:"followers"`
	StarsReceived int       `json:"stars_received"`
	TopLanguages  []string  `json:"top_languages"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// FetchAndStoreGithubStats fetches stats from GitHub API and stores in user record
func FetchAndStoreGithubStats(token string, user *models.User) error {
	client := &http.Client{}

	// 1. Get basic user info (repos, followers)
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var userData struct {
		PublicRepos int `json:"public_repos"`
		Followers   int `json:"followers"`
	}
	json.NewDecoder(resp.Body).Decode(&userData)

	// 2. Get top languages & stars (simplified for MVP)
	// Usually would iterate through repos, but let's just get total stars for now
	// GET /user/repos?sort=updated&per_page=10
	req, _ = http.NewRequest("GET", "https://api.github.com/user/repos?sort=updated&per_page=50", nil)
	req.Header.Set("Authorization", "token "+token)
	resp, err = client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		var repos []struct {
			StargazersCount int    `json:"stargazers_count"`
			Language        string `json:"language"`
		}
		json.NewDecoder(resp.Body).Decode(&repos)

		totalStars := 0
		langMap := make(map[string]int)
		for _, r := range repos {
			totalStars += r.StargazersCount
			if r.Language != "" {
				langMap[r.Language]++
			}
		}

		// Store as JSON string
		stats := GithubStatsData{
			PublicRepos:   userData.PublicRepos,
			Followers:     userData.Followers,
			StarsReceived: totalStars,
			LastUpdatedAt: time.Now(),
		}

		// Sort languages (basic)
		for l := range langMap {
			stats.TopLanguages = append(stats.TopLanguages, l)
			if len(stats.TopLanguages) > 5 {
				break
			}
		}

		statsJSON, _ := json.Marshal(stats)
		statsStr := string(statsJSON)
		user.GithubStats = &statsStr

		return database.DB.Model(user).Update("github_stats", statsStr).Error
	}

	return err
}

// SyncGithubStats endpoint
func SyncGithubStats(c *gin.Context) {
	if !database.IsFeatureEnabled(models.SettingFeatureGithubStats) {
		c.JSON(http.StatusForbidden, gin.H{"error": "GitHub Stats feature is currently disabled"})
		return
	}

	userId := c.MustGet("userId").(string)

	var user models.User
	if err := database.DB.First(&user, "id = ?", userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// This assumes the user has a linked account or we have their token stored?
	// For MVP, if they logged in via GitHub, we don't store the token long-term unless we use it now.
	// Let's assume the frontend passes the token or we have a more persistent way.
	// Actually, the user asked for: "if login with GitHub then i will directly fetch data and store in and cache it"

	c.JSON(http.StatusOK, gin.H{"message": "Sync triggered (placeholder)"})
}
