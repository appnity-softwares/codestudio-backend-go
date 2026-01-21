package seeds

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

func SeedSnippets(creator models.User) {
	log.Println("📜 Seeding Code Snippets...")

	// Fetch all user IDs to distribute snippet ownership
	var userIDs []string
	database.DB.Model(&models.User{}).Pluck("id", &userIDs)

	if len(userIDs) == 0 {
		userIDs = append(userIDs, creator.ID)
	}

	snippetTemplates := []struct {
		Title       string
		Lang        string
		Type        string
		Diff        string
		Code        string
		PreviewType string
	}{
		{"Depth First Search", "python", "ALGORITHM", "MEDIUM", "def dfs(node, visited):\n    pass", "TERMINAL"},
		{"Neumorphic Card", "html", "VISUAL", "EASY", "<div class='card'>Content</div>", "WEB_PREVIEW"},
		{"Glass Counter", "react", "EXAMPLE", "EASY", "function Counter() {}", "WEB_PREVIEW"},
		{"Binary Search", "go", "ALGORITHM", "EASY", "func search() {}", "TERMINAL"},
		{"Express Middleware", "javascript", "UTILITY", "MEDIUM", "app.use((req,res) => {})", "TERMINAL"},
		{"Quick Sort", "python", "ALGORITHM", "HARD", "def sort(arr): pass", "TERMINAL"},
		{"Tailwind Navbar", "html", "VISUAL", "EASY", "<nav class='bg-blue-500'></nav>", "WEB_PREVIEW"},
		{"Redux Toolkit", "react", "EXAMPLE", "MEDIUM", "const slice = createSlice({})", "WEB_PREVIEW"},
		{"JWT Auth", "go", "UTILITY", "HARD", "func generateToken() {}", "TERMINAL"},
		{"Custom Hook", "react", "UTILITY", "MEDIUM", "function useAuth() {}", "WEB_PREVIEW"},
		{"SQL Query Builder", "typescript", "UTILITY", "MEDIUM", "const query = select('*')", "TERMINAL"},
		{"Priority Queue", "python", "ALGORITHM", "MEDIUM", "import heapq", "TERMINAL"},
		{"GSAP Animation", "javascript", "VISUAL", "MEDIUM", "gsap.to('.box', {})", "WEB_PREVIEW"},
		{"Redis Cache", "go", "UTILITY", "EASY", "rdb.Set(ctx, 'key', 'val')", "TERMINAL"},
		{"D3 Bar Chart", "javascript", "VISUAL", "HARD", "d3.select('svg')", "WEB_PREVIEW"},
		{"Rust Vector", "rust", "EXAMPLE", "EASY", "let mut v = Vec::new();", "TERMINAL"},
		{"SwiftUI List", "swift", "EXAMPLE", "EASY", "List { Text('hi') }", "TERMINAL"},
		{"Responsive Grid", "html", "VISUAL", "EASY", "<div class='grid'></div>", "WEB_PREVIEW"},
		{"Django View", "python", "UTILITY", "MEDIUM", "def my_view(request):", "TERMINAL"},
		{"Vite Config", "typescript", "UTILITY", "EASY", "export default defineConfig({})", "TERMINAL"},
	}

	for i, t := range snippetTemplates {
		authorID := userIDs[i%len(userIDs)] // Cycle through users

		s := models.Snippet{
			ID:                  uuid.New().String(),
			Title:               t.Title,
			Description:         fmt.Sprintf("Starter implementation for %s", t.Title),
			Code:                t.Code,
			Language:            t.Lang,
			Status:              "PUBLISHED",
			Verified:            true,
			LastExecutionStatus: "SUCCESS",
			Visibility:          "PUBLIC",
			AuthorID:            authorID,
			Type:                t.Type,
			Difficulty:          t.Diff,
			PreviewType:         t.PreviewType,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		if err := database.DB.Create(&s).Error; err != nil {
			log.Printf("   ❌ Failed to create snippet %s: %v", s.Title, err)
		} else {
			log.Printf("   📝 Snippet Added: %s (%s)", s.Title, s.Language)
		}
	}
}
