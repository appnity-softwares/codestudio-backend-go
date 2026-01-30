package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

type RepoSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	Language    string `json:"language"`
	URL         string `json:"html_url"`
}

type GithubStatsSettings struct {
	ShowBio       bool `json:"show_bio"`
	ShowCompany   bool `json:"show_company"`
	ShowLocation  bool `json:"show_location"`
	ShowStats     bool `json:"show_stats"`
	ShowRepos     bool `json:"show_repos"`
	ShowLanguages bool `json:"show_languages"`
}

type GithubStatsData struct {
	Username        string              `json:"username"`
	Bio             string              `json:"bio"`
	Location        string              `json:"location"`
	Company         string              `json:"company"`
	Blog            string              `json:"blog"`
	TwitterUsername string              `json:"twitter_username"`
	Hireable        bool                `json:"hireable"`
	CreatedAt       time.Time           `json:"created_at"`
	PublicRepos     int                 `json:"public_repos"`
	Followers       int                 `json:"followers"`
	SameAs          string              `json:"html_url"`
	StarsReceived   int                 `json:"stars_received"`
	TopLanguages    []string            `json:"top_languages"`
	TopRepos        []RepoSummary       `json:"top_repos"`
	LastUpdatedAt   time.Time           `json:"last_updated_at"`
	Settings        GithubStatsSettings `json:"settings"`
}

// FetchAndStoreGithubStats fetches stats from GitHub API and stores in user record
func FetchAndStoreGithubStats(token string, user *models.User) error {
	// Preserve existing settings
	var currentSettings GithubStatsSettings
	// Default settings (all true)
	currentSettings = GithubStatsSettings{true, true, true, true, true, true}

	if user.GithubStats != nil {
		var currentData GithubStatsData
		if err := json.Unmarshal([]byte(*user.GithubStats), &currentData); err == nil {
			if !currentData.LastUpdatedAt.IsZero() {
				currentSettings = currentData.Settings
			}
		}
	}

	client := &http.Client{}

	// 1. Get basic user info (repos, followers, bio, etc)
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var userData struct {
		Login           string    `json:"login"`
		Bio             string    `json:"bio"`
		Location        string    `json:"location"`
		Company         string    `json:"company"`
		Blog            string    `json:"blog"`
		TwitterUsername string    `json:"twitter_username"`
		Hireable        bool      `json:"hireable"`
		PublicRepos     int       `json:"public_repos"`
		Followers       int       `json:"followers"`
		HtmlUrl         string    `json:"html_url"`
		CreatedAt       time.Time `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		return err
	}

	// 2. Get top languages & stars & top repos
	req, _ = http.NewRequest("GET", "https://api.github.com/user/repos?sort=pushed&per_page=100&type=owner", nil)
	req.Header.Set("Authorization", "token "+token)
	resp, err = client.Do(req)

	// Default stats
	stats := GithubStatsData{
		Username:        userData.Login,
		Bio:             userData.Bio,
		Location:        userData.Location,
		Company:         userData.Company,
		Blog:            userData.Blog,
		TwitterUsername: userData.TwitterUsername,
		Hireable:        userData.Hireable,
		CreatedAt:       userData.CreatedAt,
		PublicRepos:     userData.PublicRepos,
		Followers:       userData.Followers,
		SameAs:          userData.HtmlUrl,
		LastUpdatedAt:   time.Now(),
		Settings:        currentSettings,
	}

	if err == nil {
		defer resp.Body.Close()
		var repos []struct {
			Name            string `json:"name"`
			Description     string `json:"description"`
			StargazersCount int    `json:"stargazers_count"`
			Language        string `json:"language"`
			HtmlUrl         string `json:"html_url"`
			Fork            bool   `json:"fork"`
		}
		json.NewDecoder(resp.Body).Decode(&repos)

		totalStars := 0
		langMap := make(map[string]int)
		var repoSummaries []RepoSummary

		for _, r := range repos {
			if r.Fork {
				continue
			}

			totalStars += r.StargazersCount
			if r.Language != "" {
				langMap[r.Language]++
			}

			repoSummaries = append(repoSummaries, RepoSummary{
				Name:        r.Name,
				Description: r.Description,
				Stars:       r.StargazersCount,
				Language:    r.Language,
				URL:         r.HtmlUrl,
			})
		}
		stats.StarsReceived = totalStars

		// Sort languages
		for l := range langMap {
			stats.TopLanguages = append(stats.TopLanguages, l)
		}
		if len(stats.TopLanguages) > 5 {
			stats.TopLanguages = stats.TopLanguages[:5]
		}

		// Sort Repos by Stars
		for i := 0; i < len(repoSummaries); i++ {
			for j := i + 1; j < len(repoSummaries); j++ {
				if repoSummaries[j].Stars > repoSummaries[i].Stars {
					repoSummaries[i], repoSummaries[j] = repoSummaries[j], repoSummaries[i]
				}
			}
		}
		if len(repoSummaries) > 3 {
			stats.TopRepos = repoSummaries[:3]
		} else {
			stats.TopRepos = repoSummaries
		}
	}

	statsJSON, _ := json.Marshal(stats)
	statsStr := string(statsJSON)
	user.GithubStats = &statsStr

	return database.DB.Model(user).Update("github_stats", statsStr).Error
}

// SyncGithubStats endpoint
func SyncGithubStats(c *gin.Context) {
	// For MVP: We don't store GitHub tokens, so we cannot sync in background.
	// We return a status code properly indicating this limitation.
	c.JSON(http.StatusOK, gin.H{"message": "Please reconnect via the 'Connect GitHub' button to refresh stats."})
}

// UpdateGithubStatsSettings updates the visibility preferences
func UpdateGithubStatsSettings(c *gin.Context) {
	userId := c.MustGet("userId").(string)

	var input GithubStatsSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.GithubStats == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No GitHub stats found to configure"})
		return
	}

	var currentData GithubStatsData
	if err := json.Unmarshal([]byte(*user.GithubStats), &currentData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse existing stats"})
		return
	}

	// Update settings
	currentData.Settings = input

	statsJSON, _ := json.Marshal(currentData)
	statsStr := string(statsJSON)
	user.GithubStats = &statsStr

	if err := database.DB.Model(&user).Update("github_stats", statsStr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated", "settings": input})
}
