package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pushp314/devconnect-backend/pkg/logger"
	"golang.org/x/time/rate"
)

// IPRateLimiter manages rate limiters for each IP
type IPRateLimiter struct {
	ips   map[string]*rateLimiterEntry
	mu    sync.RWMutex
	r     rate.Limit
	burst int
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter
// r = requests per second, burst = max burst size
func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	rl := &IPRateLimiter{
		ips:   make(map[string]*rateLimiterEntry),
		r:     r,
		burst: burst,
	}

	// Cleanup old entries every minute
	go rl.cleanup()

	return rl
}

func (rl *IPRateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, entry := range rl.ips {
			if time.Since(entry.lastSeen) > 3*time.Minute {
				delete(rl.ips, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// GetLimiter returns the rate limiter for the given IP
func (rl *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.ips[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.r, rl.burst)
		rl.ips[ip] = &rateLimiterEntry{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// Pre-configured rate limiters for different endpoints
var (
	// Auth endpoints: 20 requests per minute
	AuthLimiter = NewIPRateLimiter(rate.Limit(20.0/60.0), 10)

	// Code execution: 60 requests per minute (1/sec)
	ExecuteLimiter = NewIPRateLimiter(rate.Limit(1.0), 5)

	// General API: 600 requests per minute (10/sec)
	GeneralLimiter = NewIPRateLimiter(rate.Limit(10.0), 50)

	// Contest submission: 20 per minute
	SubmitLimiter = NewIPRateLimiter(rate.Limit(20.0/60.0), 5)

	// Chat messages: 30 per minute (prevents spam, allows normal conversation)
	ChatLimiter = NewIPRateLimiter(rate.Limit(30.0/60.0), 10)
)

// RateLimitMiddleware creates a rate limiting middleware with a custom limiter
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.GetLimiter(ip)

		if !l.Allow() {
			logger.Warn().
				Str("ip", ip).
				Str("path", c.Request.URL.Path).
				Msg("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests",
				"message": "Rate limit exceeded. Please slow down.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimit is a convenience wrapper for auth endpoints
func AuthRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(AuthLimiter)
}

// ExecuteRateLimit is for code execution endpoints
func ExecuteRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(ExecuteLimiter)
}

// GeneralRateLimit is for general API endpoints
func GeneralRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(GeneralLimiter)
}

// SubmitRateLimit is for contest submission endpoints
func SubmitRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(SubmitLimiter)
}

// ChatRateLimit is for chat message endpoints
func ChatRateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(ChatLimiter)
}
