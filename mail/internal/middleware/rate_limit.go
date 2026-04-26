package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
	"github.com/gin-gonic/gin"
)

// bucket tracks how many requests have been made in the current window.
type bucket struct {
	count   int
	resetAt time.Time
}

// InMemoryLimiter is a simple in-memory rate limiter.  It is suitable for a
// single-instance MVP and should be replaced by Redis when scaling out.
type InMemoryLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

// NewInMemoryLimiter creates a new in-memory rate limiter.
func NewInMemoryLimiter() *InMemoryLimiter {
	return &InMemoryLimiter{buckets: make(map[string]*bucket)}
}

// Allow reports whether the request identified by key is permitted.  If this
// is the first request in a window, a new bucket is created.  If the window
// has expired, the bucket is reset.
func (l *InMemoryLimiter) Allow(key string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok || now.After(b.resetAt) {
		l.buckets[key] = &bucket{count: 1, resetAt: now.Add(window)}
		return true
	}
	if b.count >= limit {
		return false
	}
	b.count++
	return true
}

// RateLimitByIP returns middleware that limits requests by client IP.
func RateLimitByIP(limiter *InMemoryLimiter, limit int, window time.Duration, auditRepo *repo.AuditRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}
		key := "ip:" + c.ClientIP()
		if !limiter.Allow(key, limit, window) {
			logRateLimit(c, auditRepo, key)
			c.String(http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitByUser returns middleware that limits requests by authenticated
// user (or operator).  It must be applied *after* an auth middleware so that
// "user" is present in the Gin context.
func RateLimitByUser(limiter *InMemoryLimiter, limit int, window time.Duration, auditRepo *repo.AuditRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}
		var key string
		if userVal, ok := c.Get("user"); ok {
			if u, ok := userVal.(*models.User); ok {
				key = fmt.Sprintf("user:%d", u.ID)
			} else {
				key = "user:unknown"
			}
		} else {
			key = "user:anon:" + c.ClientIP()
		}
		if !limiter.Allow(key, limit, window) {
			logRateLimit(c, auditRepo, key)
			c.String(http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func logRateLimit(c *gin.Context, auditRepo *repo.AuditRepo, key string) {
	if auditRepo == nil {
		return
	}
	var actorID *int
	if userVal, ok := c.Get("user"); ok {
		if u, ok := userVal.(*models.User); ok && u.ID != 0 {
			actorID = &u.ID
		}
	}
	_ = auditRepo.Log(c.Request.Context(), &models.AuditLog{
		ActorUserID: actorID,
		Action:      "rate_limit_exceeded",
		EntityType:  "security",
		Payload: map[string]interface{}{
			"ip":   c.ClientIP(),
			"path": c.Request.URL.Path,
			"key":  key,
		},
	})
}
