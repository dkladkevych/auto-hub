package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter provides in-memory per-IP rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	limiters map[string]*rate.Limiter
	maxRPS   rate.Limit
	burst    int
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		maxRPS:   rate.Limit(rps),
		burst:    burst,
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	lim, ok := rl.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(rl.maxRPS, rl.burst)
		rl.limiters[ip] = lim
	}
	return lim
}

func clientIP(r *http.Request) string {
	ip := r.RemoteAddr
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		ip = strings.Split(xf, ",")[0]
	}
	return strings.TrimSpace(ip)
}

// RateLimit middleware limits requests per IP.
func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		lim := rl.getLimiter(clientIP(c.Request))
		if !lim.Allow() {
			c.String(http.StatusTooManyRequests, "Rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}
