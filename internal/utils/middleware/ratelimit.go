package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/port/outbound"
)

const (
	// RateLimitRemaining is the header for remaining requests.
	RateLimitRemaining = "X-RateLimit-Remaining"
	// RateLimitLimit is the header for the limit.
	RateLimitLimit = "X-RateLimit-Limit"
	// RateLimitReset is the header for reset time.
	RateLimitReset = "X-RateLimit-Reset"
	// RetryAfter is the header for retry time.
	RetryAfter = "Retry-After"
)

// RateLimitConfig holds rate limit configuration.
type RateLimitConfig struct {
	// Limit is the maximum number of requests.
	Limit int
	// Window is the time window.
	Window time.Duration
	// KeyFunc generates the rate limit key from request.
	// Default uses client IP.
	KeyFunc func(*gin.Context) string
	// SkipFunc determines if the request should skip rate limiting.
	SkipFunc func(*gin.Context) bool
}

// DefaultRateLimitConfig returns the default rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limit:  100,
		Window: time.Minute,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
		SkipFunc: nil,
	}
}

// RateLimit returns a middleware that limits requests using the given limiter.
func RateLimit(limiter outbound.RateLimiterPort, cfg RateLimitConfig) gin.HandlerFunc {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *gin.Context) string {
			return c.ClientIP()
		}
	}

	return func(c *gin.Context) {
		// Skip if limiter is nil
		if limiter == nil {
			c.Next()
			return
		}

		// Check skip function
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}

		key := cfg.KeyFunc(c)
		ctx := c.Request.Context()

		// Check rate limit
		allowed, err := limiter.Allow(ctx, key, cfg.Limit, cfg.Window)
		if err != nil {
			// On error, allow the request but log it
			c.Next()
			return
		}

		// Get remaining count
		remaining, _ := limiter.GetRemaining(ctx, key, cfg.Limit, cfg.Window)

		// Set headers
		c.Header(RateLimitLimit, strconv.Itoa(cfg.Limit))
		c.Header(RateLimitRemaining, strconv.Itoa(remaining))
		c.Header(RateLimitReset, strconv.FormatInt(time.Now().Add(cfg.Window).Unix(), 10))

		if !allowed {
			c.Header(RetryAfter, strconv.Itoa(int(cfg.Window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests, please try again later",
				},
			})
			return
		}

		c.Next()
	}
}

// RateLimitByIP returns a rate limiter that limits by IP address.
func RateLimitByIP(limiter outbound.RateLimiterPort, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimit(limiter, RateLimitConfig{
		Limit:  limit,
		Window: window,
		KeyFunc: func(c *gin.Context) string {
			return "ip:" + c.ClientIP()
		},
	})
}

// RateLimitByUser returns a rate limiter that limits by user ID.
// Falls back to IP if user is not authenticated.
func RateLimitByUser(limiter outbound.RateLimiterPort, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimit(limiter, RateLimitConfig{
		Limit:  limit,
		Window: window,
		KeyFunc: func(c *gin.Context) string {
			userID := GetUserID(c)
			if userID.String() != "00000000-0000-0000-0000-000000000000" {
				return "user:" + userID.String()
			}
			return "ip:" + c.ClientIP()
		},
	})
}

// RateLimitByEndpoint returns a rate limiter that limits by endpoint and IP.
func RateLimitByEndpoint(limiter outbound.RateLimiterPort, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimit(limiter, RateLimitConfig{
		Limit:  limit,
		Window: window,
		KeyFunc: func(c *gin.Context) string {
			return fmt.Sprintf("endpoint:%s:%s:%s", c.Request.Method, c.FullPath(), c.ClientIP())
		},
	})
}
