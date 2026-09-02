package http

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a production-ready CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

// CORSMiddleware returns a Gin middleware that sets CORS headers.
func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		for _, o := range cfg.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
			if cfg.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if cfg.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*tokenBucket
	rate     int           // tokens per interval
	interval time.Duration // refill interval
	burst    int           // max tokens
}

type tokenBucket struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a rate limiter with the given rate and burst.
func NewRateLimiter(rate int, interval time.Duration, burst int) *RateLimiter {
	return &RateLimiter{
		clients:  make(map[string]*tokenBucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}
}

// Allow checks if a request from the given key is allowed.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	bucket, exists := r.clients[key]
	if !exists {
		r.clients[key] = &tokenBucket{tokens: r.burst - 1, lastCheck: now}
		return true
	}

	// Refill tokens
	elapsed := now.Sub(bucket.lastCheck)
	tokensToAdd := int(elapsed.Seconds() * float64(r.rate) / r.interval.Seconds())
	bucket.tokens += tokensToAdd
	if bucket.tokens > r.burst {
		bucket.tokens = r.burst
	}
	bucket.lastCheck = now

	if bucket.tokens <= 0 {
		return false
	}
	bucket.tokens--
	return true
}

// RateLimitMiddleware returns a Gin middleware that rate-limits by client IP.
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "too many requests",
				},
			})
			return
		}
		c.Next()
	}
}

// RequestLogger returns a Gin middleware that logs all requests with user, IP, and duration.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		user := c.GetHeader("X-User-Id")
		if user == "" {
			user = "anonymous"
		}

		log.Printf("[API] %s %s %s %d %v user=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.Writer.Status(),
			duration,
			user,
		)
	}
}
