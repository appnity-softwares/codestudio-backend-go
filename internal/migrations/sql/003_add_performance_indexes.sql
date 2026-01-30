-- =============================================================================
-- P0 Performance Fix: Add Missing Database Indexes
-- =============================================================================
-- 
-- IMPORTANT: This script uses CREATE INDEX CONCURRENTLY which:
-- 1. Does NOT lock the table for reads
-- 2. Does NOT block INSERT/UPDATE/DELETE operations
-- 3. Takes longer but is SAFE for production
--
-- REQUIREMENTS:
-- - PostgreSQL 9.6+ (CONCURRENTLY support)
-- - Must run OUTSIDE of a transaction block (no BEGIN/COMMIT wrapper)
-- - autocommit must be ON
--
-- USAGE:
--   psql -h <host> -U <user> -d <database> -f 003_add_performance_indexes.sql
--
-- IDEMPOTENT: Safe to run multiple times (uses IF NOT EXISTS)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Index 1: Snippet Reactions Lookup
-- -----------------------------------------------------------------------------
-- Table: "SnippetReaction"
-- Columns: (user_id, snippet_id)
--
-- Optimizes queries like:
--   SELECT * FROM "SnippetReaction" WHERE user_id = $1 AND snippet_id = $2
--   SELECT snippet_id, reaction FROM "SnippetReaction" WHERE user_id = $1 AND snippet_id IN ($2, $3, ...)
--
-- Expected improvement: 10x+ faster reaction lookups in feed population
-- -----------------------------------------------------------------------------
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snippet_reactions_user_snippet 
ON "SnippetReaction" (user_id, snippet_id);

-- -----------------------------------------------------------------------------
-- Index 2: Submission Duplicate Detection
-- -----------------------------------------------------------------------------
-- Table: submissions
-- Columns: (problem_id, code_hash, user_id)
--
-- Optimizes queries like:
--   SELECT * FROM submissions WHERE problem_id = $1 AND code_hash = $2 AND user_id != $3
--
-- Expected improvement: 100x+ faster plagiarism detection during contest submissions
-- Critical for anti-cheat functionality under load
-- -----------------------------------------------------------------------------
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_duplicate_check 
ON submissions (problem_id, code_hash, user_id);

-- -----------------------------------------------------------------------------
-- Index 3: Feed Filtering (Status + CreatedAt)
-- -----------------------------------------------------------------------------
-- Table: "Snippet"
-- Columns: (status, "createdAt" DESC)
--
-- Optimizes queries like:
--   SELECT * FROM "Snippet" WHERE status = 'PUBLISHED' ORDER BY "createdAt" DESC LIMIT 20
--   SELECT * FROM "Snippet" WHERE status = 'PUBLISHED' AND visibility = 'public' ORDER BY "createdAt" DESC
--
-- Expected improvement: 50x+ faster feed queries as snippet count grows
-- Index is ordered DESC to match common query patterns
-- -----------------------------------------------------------------------------
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snippets_feed 
ON "Snippet" (status, "createdAt" DESC);

-- =============================================================================
-- Verification: Check indexes were created successfully
-- =============================================================================
-- Run this after execution to verify:
--
-- SELECT indexname, indexdef 
-- FROM pg_indexes 
-- WHERE tablename IN ('SnippetReaction', 'submissions', 'Snippet')
--   AND indexname LIKE 'idx_%';
--
-- Expected output: 3 rows with the new indexes
-- =============================================================================
