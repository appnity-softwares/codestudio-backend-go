package handlers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

const BaseURL = "https://codestudio.appnity.cloud"

// Sitemap Cache
var (
	sitemapCache     []byte
	sitemapRefreshed time.Time
	sitemapMutex     sync.RWMutex
	cacheDuration    = 6 * time.Hour
)

// SitemapEntry represents a single URL entry in the sitemap
type SitemapEntry struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	LastMod    string   `xml:"lastmod,omitempty"`
	ChangeFreq string   `xml:"changefreq,omitempty"`
	Priority   string   `xml:"priority,omitempty"`
}

// URLSet is the root element of the sitemap
type URLSet struct {
	XMLName xml.Name       `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []SitemapEntry `xml:"url"`
}

// GenerateSitemap handles the dynamic sitemap generation with caching
func GenerateSitemap(c *gin.Context) {
	// 1. Check Cache
	sitemapMutex.RLock()
	if sitemapCache != nil && time.Since(sitemapRefreshed) < cacheDuration {
		c.Header("Content-Type", "application/xml")
		c.Writer.Write(sitemapCache)
		sitemapMutex.RUnlock()
		return
	}
	sitemapMutex.RUnlock()

	var urls []SitemapEntry

	// 2. Static Pages (Public)
	staticPages := []string{"", "/snippets", "/arena", "/changelog"}
	for _, p := range staticPages {
		urls = append(urls, SitemapEntry{
			Loc:        BaseURL + p,
			ChangeFreq: "daily",
			Priority:   "0.8",
		})
	}

	// 3. Snippets (Public only)
	var snippets []models.Snippet
	database.DB.Select("id, updated_at").Where("visibility = ?", "PUBLIC").Order("created_at desc").Limit(2000).Find(&snippets)
	for _, s := range snippets {
		urls = append(urls, SitemapEntry{
			Loc:        fmt.Sprintf("%s/snippet/%s", BaseURL, s.ID),
			LastMod:    s.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.6",
		})
	}

	// 4. Contests (Public + Live/Upcoming/Ended)
	var events []models.Event
	database.DB.Select("id, updated_at").Where("status IN ?", []string{"LIVE", "UPCOMING", "ENDED", "FROZEN"}).Find(&events)
	for _, e := range events {
		urls = append(urls, SitemapEntry{
			Loc:        fmt.Sprintf("%s/contest/%s", BaseURL, e.ID),
			LastMod:    e.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "daily",
			Priority:   "0.7",
		})
	}

	// 5. User Profiles (Public only)
	var users []models.User
	database.DB.Select("username, created_at").Where("visibility = ?", "PUBLIC").Limit(1000).Find(&users)
	for _, u := range users {
		urls = append(urls, SitemapEntry{
			Loc:        fmt.Sprintf("%s/u/%s", BaseURL, u.Username),
			LastMod:    u.CreatedAt.Format("2006-01-02"),
			ChangeFreq: "monthly",
			Priority:   "0.5",
		})
	}

	urlSet := URLSet{URLs: urls}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	finalXML := []byte(xml.Header + string(output))

	// Update Cache
	sitemapMutex.Lock()
	sitemapCache = finalXML
	sitemapRefreshed = time.Now()
	sitemapMutex.Unlock()

	c.Header("Content-Type", "application/xml")
	c.Writer.Write(finalXML)
}

// GenerateRobotsTXT returns the robots.txt file
func GenerateRobotsTXT(c *gin.Context) {
	robots := `User-agent: *
Allow: /
Disallow: /login
Disallow: /register
Disallow: /admin
Disallow: /chat
Disallow: /api
Disallow: /settings
Disallow: /auth

Sitemap: https://codestudio.appnity.cloud/sitemap.xml`

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, robots)
}
