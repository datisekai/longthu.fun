package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// MaxRequests per window
	MaxRequests = 60
	// Window size
	Window = time.Minute
)

// IPTracker tracks requests per IP.
type IPTracker struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

func New() *IPTracker {
	return &IPTracker{requests: make(map[string][]time.Time)}
}

// Allow returns true if the request should be allowed.
func (t *IPTracker) Allow(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-Window)

	// Filter old requests
	var valid []time.Time
	for _, ts := range t.requests[ip] {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	if len(valid) >= MaxRequests {
		t.requests[ip] = valid
		return false
	}

	t.requests[ip] = append(valid, now)
	return true
}

// Remaining returns how many requests are left for this IP.
func (t *IPTracker) Remaining(ip string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-Window)
	count := 0
	for _, ts := range t.requests[ip] {
		if ts.After(cutoff) {
			count++
		}
	}
	return MaxRequests - count
}

// Global tracker
var global = New()

// Middleware returns a Gin middleware that rate-limits by IP.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !global.Allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"title":      "Quá nhiều request",
				"retryAfter": 60,
			})
			return
		}
		c.Next()
	}
}
