package migrations

import (
	"gorm.io/gorm"
)

// Migration003AddPerformanceIndexes adds missing indexes for hot-path queries
// This addresses P0 database performance issues identified in the production audit:
// 1. Snippet reactions lookup (user_id, snippet_id)
// 2. Submission duplicate detection (problem_id, code_hash, user_id)
// 3. Feed filtering (status, createdAt)
//
// Uses CREATE INDEX CONCURRENTLY to avoid table locks in production.
// All indexes are idempotent (IF NOT EXISTS) for safe re-runs.
func Migration003AddPerformanceIndexes() Migration {
	return Migration{
		ID:   "003_add_performance_indexes",
		Name: "Add performance indexes for hot-path queries",
		Up: func(db *gorm.DB) error {
			// Note: PostgreSQL's CREATE INDEX CONCURRENTLY cannot run inside a transaction.
			// We need to use raw Exec outside of GORM's transaction wrapper.
			// Since the migrator wraps Up() in a transaction, we use individual
			// CREATE INDEX IF NOT EXISTS statements (without CONCURRENTLY) for now.
			// For production deployments, run the concurrent version manually.

			// Index 1: Snippet Reactions Lookup
			// Optimizes: WHERE user_id = ? AND snippet_id = ?
			// Query pattern: Check if user has reacted, get user's reaction for a snippet
			idx1 := `
				CREATE INDEX IF NOT EXISTS idx_snippet_reactions_user_snippet 
				ON "SnippetReaction" (user_id, snippet_id)
			`
			if err := db.Exec(idx1).Error; err != nil {
				return err
			}

			// Index 2: Submission Duplicate Detection
			// Optimizes: WHERE problem_id = ? AND code_hash = ? AND user_id != ?
			// Query pattern: Anti-cheat duplicate code detection
			idx2 := `
				CREATE INDEX IF NOT EXISTS idx_submissions_duplicate_check 
				ON submissions (problem_id, code_hash, user_id)
			`
			if err := db.Exec(idx2).Error; err != nil {
				return err
			}

			// Index 3: Feed Filtering (Status + CreatedAt)
			// Optimizes: WHERE status = 'PUBLISHED' ORDER BY createdAt DESC
			// Query pattern: Main feed queries, trending, new snippets
			idx3 := `
				CREATE INDEX IF NOT EXISTS idx_snippets_feed 
				ON "Snippet" (status, "createdAt" DESC)
			`
			if err := db.Exec(idx3).Error; err != nil {
				return err
			}

			return nil
		},
		Down: func(db *gorm.DB) error {
			// Drop indexes in reverse order
			if err := db.Exec(`DROP INDEX IF EXISTS idx_snippets_feed`).Error; err != nil {
				return err
			}
			if err := db.Exec(`DROP INDEX IF EXISTS idx_submissions_duplicate_check`).Error; err != nil {
				return err
			}
			if err := db.Exec(`DROP INDEX IF EXISTS idx_snippet_reactions_user_snippet`).Error; err != nil {
				return err
			}
			return nil
		},
	}
}
