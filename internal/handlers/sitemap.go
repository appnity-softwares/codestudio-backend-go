package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

// GenerateSitemap generates a dynamic sitemap.xml
func GenerateSitemap(c *gin.Context) {
	c.Header("Content-Type", "application/xml")

	var snippets []models.Snippet
	database.DB.Select("id, updated_at").Where("visibility = ?", "public").Find(&snippets)

	var users []models.User
	database.DB.Select("username, updated_at").Where("visibility = ?", "PUBLIC").Find(&users)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://codestudio.dev/</loc>
		<changefreq>daily</changefreq>
		<priority>1.0</priority>
	</url>
	<url>
		<loc>https://codestudio.dev/auth/signin</loc>
		<changefreq>monthly</changefreq>
		<priority>0.8</priority>
	</url>
	<url>
		<loc>https://codestudio.dev/changelog</loc>
		<changefreq>weekly</changefreq>
		<priority>0.7</priority>
	</url>`

	// Add Snippets
	for _, s := range snippets {
		xml += fmt.Sprintf(`
	<url>
		<loc>https://codestudio.dev/snippets/%s</loc>
		<lastmod>%s</lastmod>
		<changefreq>weekly</changefreq>
		<priority>0.6</priority>
	</url>`, s.ID, s.UpdatedAt.Format(time.RFC3339))
	}

	// Add Users
	for _, u := range users {
		xml += fmt.Sprintf(`
	<url>
		<loc>https://codestudio.dev/u/%s</loc>
		<lastmod>%s</lastmod>
		<changefreq>weekly</changefreq>
		<priority>0.5</priority>
	</url>`, u.Username, u.UpdatedAt.Format(time.RFC3339))
	}

	xml += `
</urlset>`

	c.String(http.StatusOK, xml)
}

// GenerateRobotsTXT generates a dynamic robots.txt
func GenerateRobotsTXT(c *gin.Context) {
	c.Header("Content-Type", "text/plain")

	txt := `User-agent: *
Allow: /
Disallow: /admin/
Disallow: /settings/
Disallow: /api/

Sitemap: https://codestudio.dev/sitemap.xml
`
	c.String(http.StatusOK, txt)
}
