package services

import (
	"sort"
	"sync"
	"time"

	"github.com/pushp314/devconnect-backend/internal/database"
	"github.com/pushp314/devconnect-backend/internal/models"
)

type LeaderboardEntry struct {
	Rank         int                    `json:"rank"`
	UserID       string                 `json:"userId"`
	Username     string                 `json:"username"`
	Name         string                 `json:"name"`
	Avatar       string                 `json:"avatar"`
	TrustScore   int                    `json:"trustScore"`
	TotalScore   int                    `json:"totalScore"`
	SolvedCount  int                    `json:"solvedCount"`
	TotalTime    float64                `json:"totalTime"`    // Minutes
	LastSubmitAt time.Time              `json:"lastSubmitAt"` // For tie-breaking
	Problems     map[string]ProblemStat `json:"problems"`     // ProblemID -> Stat
	Status       string                 `json:"status"`       // NORMAL, UNDER_REVIEW, DISQUALIFIED
	FlagsCount   int                    `json:"flagsCount"`
}

type ProblemStat struct {
	Status    string  `json:"status"` // AC, WA, PENDING
	Attempts  int     `json:"attempts"`
	TimeTaken float64 `json:"timeTaken"` // Minutes (including penalty)
	Penalty   int     `json:"penalty"`
}

// In-memory cache: EventID -> {Entries, Expiry}
type cachedLeaderboard struct {
	Entries   []LeaderboardEntry
	ExpiresAt time.Time
}

var (
	leaderboardCache = make(map[string]cachedLeaderboard)
	lbMutex          sync.RWMutex
	lbTTL            = 10 * time.Second // Refresh every 10s max
)

// InvalidateLeaderboardCache clears cache for an event (call on new submission)
func InvalidateLeaderboardCache(eventID string) {
	lbMutex.Lock()
	defer lbMutex.Unlock()
	delete(leaderboardCache, eventID)
}

// GetLeaderboard calculates or returns cached leaderboard
// if asAdmin is true, it ignores freeze
func GetLeaderboard(eventID string, asAdmin bool) ([]LeaderboardEntry, error) {
	// 1. Check Cache (Only for public view)
	if !asAdmin {
		lbMutex.RLock()
		if cached, ok := leaderboardCache[eventID]; ok {
			if time.Now().Before(cached.ExpiresAt) {
				lbMutex.RUnlock()
				return cached.Entries, nil
			}
		}
		lbMutex.RUnlock()
	}

	// 2. Fetch Data
	var event models.Event
	if err := database.DB.Preload("Problems").First(&event, "id = ?", eventID).Error; err != nil {
		return nil, err
	}

	// Freeze Check
	cutoffTime := time.Now()
	// If not admin, and frozen, set cutoff to FreezeTime
	if !asAdmin && event.FreezeTime != nil && time.Now().After(*event.FreezeTime) {
		cutoffTime = *event.FreezeTime
	}

	// Fetch all submissions for this event up to cutoff
	var submissions []models.Submission
	if err := database.DB.Preload("User").Preload("Flags").
		Where("event_id = ? AND created_at <= ?", eventID, cutoffTime).
		Order("created_at asc"). // Process chronological
		Find(&submissions).Error; err != nil {
		return nil, err
	}

	// Map to aggregated stats
	// UserID -> Entry
	userMap := make(map[string]*LeaderboardEntry)

	// Pre-fill registered users? Or just those who submitted?
	// Usually leaderboard shows only those who submitted.

	// Helper to get penalty for problem
	problemPenalty := make(map[string]int)
	for _, p := range event.Problems {
		penalty := p.Penalty
		if penalty == 0 {
			penalty = 10 // Default
		}
		problemPenalty[p.ID] = penalty
	}

	for _, sub := range submissions {
		if userMap[sub.UserID] == nil {
			status := "NORMAL"
			// Check user trust/flags
			if sub.User.TrustScore < 50 {
				status = "UNDER_REVIEW"
			}
			userMap[sub.UserID] = &LeaderboardEntry{
				UserID:     sub.UserID,
				Username:   sub.User.Username,
				Name:       sub.User.Name,
				Avatar:     sub.User.Image,
				TrustScore: sub.User.TrustScore,
				Problems:   make(map[string]ProblemStat),
				Status:     status,
			}
		}

		entry := userMap[sub.UserID]

		// Anti-Cheat: If user has unresolved flags, mark under review
		if len(sub.Flags) > 0 {
			entry.FlagsCount += len(sub.Flags)
			if entry.Status != "DISQUALIFIED" {
				entry.Status = "UNDER_REVIEW"
			}
		}

		// Problem Logic
		probStat := entry.Problems[sub.ProblemID]

		// If already AC, ignore subsequent submissions
		if probStat.Status == string(models.SubStatusAC) {
			continue
		}

		probStat.Attempts++

		if sub.Status == models.SubStatusAC {
			probStat.Status = string(models.SubStatusAC)

			// Calculate Time: (SubmitTime - StartTime) in minutes
			submissionTime := sub.CreatedAt.Sub(event.StartTime).Minutes()
			if submissionTime < 0 {
				submissionTime = 0
			}

			// Calculate Penalty Time
			penaltyMinutes := float64((probStat.Attempts - 1) * problemPenalty[sub.ProblemID])

			probStat.TimeTaken = submissionTime + penaltyMinutes
			probStat.Penalty = int(penaltyMinutes)

			// Update User Totals
			entry.SolvedCount++

			// Points? Need to look up problem points.
			// Ideally we fetch problems and map ID to Points.
			// For MVP efficiency let's do a lookup or include points in problemPenalty map struct
			// Let's assume we have it. Re-looping event.Problems is fast.
			points := 0
			for _, p := range event.Problems {
				if p.ID == sub.ProblemID {
					points = p.Points
					break
				}
			}
			entry.TotalScore += points
			entry.TotalTime += probStat.TimeTaken
			entry.LastSubmitAt = sub.CreatedAt

		} else {
			// WA, TLE, RE, CE -> Just attempts incremented above
			probStat.Status = string(sub.Status)
		}

		entry.Problems[sub.ProblemID] = probStat
	}

	// Convers map to slice
	var leaderboard []LeaderboardEntry
	for _, entry := range userMap {
		leaderboard = append(leaderboard, *entry)
	}

	// 3. Sort
	sort.Slice(leaderboard, func(i, j int) bool {
		// 1. Solved Count DESC
		if leaderboard[i].SolvedCount != leaderboard[j].SolvedCount {
			return leaderboard[i].SolvedCount > leaderboard[j].SolvedCount
		}
		// 2. Total Score DESC
		if leaderboard[i].TotalScore != leaderboard[j].TotalScore {
			return leaderboard[i].TotalScore > leaderboard[j].TotalScore
		}
		// 3. Total Time ASC
		if leaderboard[i].TotalTime != leaderboard[j].TotalTime {
			return leaderboard[i].TotalTime < leaderboard[j].TotalTime
		}
		// 4. Last Submit Time ASC (Earliest best submission wins tie)
		return leaderboard[i].LastSubmitAt.Before(leaderboard[j].LastSubmitAt)
	})

	// Assign Ranks
	for i := range leaderboard {
		leaderboard[i].Rank = i + 1
	}

	// 4. Cache (Only cache if public view, or if frozen it is static anyway)
	if !asAdmin {
		lbMutex.Lock()
		leaderboardCache[eventID] = cachedLeaderboard{
			Entries:   leaderboard,
			ExpiresAt: time.Now().Add(lbTTL),
		}
		lbMutex.Unlock()
	}

	return leaderboard, nil
}
