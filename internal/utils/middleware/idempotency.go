package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// IdempotencyKeyHeader is the header for idempotency key.
	IdempotencyKeyHeader = "Idempotency-Key"
	// idempotencyKeyPrefix is the Redis key prefix.
	idempotencyKeyPrefix = "idempotency:"
	// defaultIdempotencyTTL is the default TTL for idempotency keys.
	defaultIdempotencyTTL = 24 * time.Hour
)

// IdempotencyConfig holds idempotency middleware configuration.
type IdempotencyConfig struct {
	// TTL is the time to live for idempotency keys.
	TTL time.Duration
	// Methods are the HTTP methods to apply idempotency check.
	// Default: POST, PUT, PATCH
	Methods []string
	// SkipFunc determines if the request should skip idempotency check.
	SkipFunc func(*gin.Context) bool
}

// DefaultIdempotencyConfig returns the default idempotency configuration.
func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		TTL:     defaultIdempotencyTTL,
		Methods: []string{"POST", "PUT", "PATCH"},
	}
}

// idempotencyResponse stores the cached response.
type idempotencyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// idempotencyResponseWriter wraps gin.ResponseWriter to capture the response.
type idempotencyResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *idempotencyResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Idempotency returns a middleware that ensures request idempotency.
// Requires Redis for storing idempotency keys and responses.
func Idempotency(redis goredis.UniversalClient, cfg IdempotencyConfig) gin.HandlerFunc {
	if cfg.TTL == 0 {
		cfg.TTL = defaultIdempotencyTTL
	}
	if len(cfg.Methods) == 0 {
		cfg.Methods = []string{"POST", "PUT", "PATCH"}
	}

	methodSet := make(map[string]bool)
	for _, m := range cfg.Methods {
		methodSet[m] = true
	}

	return func(c *gin.Context) {
		// Skip if Redis is nil
		if redis == nil {
			c.Next()
			return
		}

		// Skip if method is not in the list
		if !methodSet[c.Request.Method] {
			c.Next()
			return
		}

		// Skip if custom skip function returns true
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}

		// Get idempotency key from header
		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
		if idempotencyKey == "" {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// Generate cache key (includes path and method for extra safety)
		cacheKey := generateIdempotencyKey(c, idempotencyKey)

		// Try to get cached response
		cachedResp, err := getCachedResponse(ctx, redis, cacheKey)
		if err == nil && cachedResp != nil {
			// Return cached response
			for k, v := range cachedResp.Headers {
				c.Header(k, v)
			}
			c.Data(cachedResp.StatusCode, c.Writer.Header().Get("Content-Type"), cachedResp.Body)
			c.Abort()
			return
		}

		// Check if request is in progress (lock)
		lockKey := cacheKey + ":lock"
		locked, err := redis.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
		if err != nil {
			c.Next()
			return
		}

		if !locked {
			// Request in progress, return conflict
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "REQUEST_IN_PROGRESS",
					"message": "A request with this idempotency key is already being processed",
				},
			})
			return
		}

		// Clean up lock when done
		defer redis.Del(ctx, lockKey)

		// Wrap response writer to capture response
		respWriter := &idempotencyResponseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = respWriter

		// Process request
		c.Next()

		// Cache the response
		if c.Writer.Status() >= 200 && c.Writer.Status() < 500 {
			headers := make(map[string]string)
			for k := range c.Writer.Header() {
				headers[k] = c.Writer.Header().Get(k)
			}

			resp := &idempotencyResponse{
				StatusCode: c.Writer.Status(),
				Headers:    headers,
				Body:       respWriter.body.Bytes(),
			}

			cacheResponse(ctx, redis, cacheKey, resp, cfg.TTL)
		}
	}
}

// generateIdempotencyKey generates a cache key from the request.
func generateIdempotencyKey(c *gin.Context, idempotencyKey string) string {
	// Include method and path in key for extra safety
	hash := sha256.Sum256([]byte(c.Request.Method + ":" + c.FullPath() + ":" + idempotencyKey))
	return idempotencyKeyPrefix + hex.EncodeToString(hash[:])
}

// getCachedResponse retrieves a cached response from Redis.
func getCachedResponse(ctx context.Context, redis goredis.UniversalClient, key string) (*idempotencyResponse, error) {
	data, err := redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var resp idempotencyResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// cacheResponse stores a response in Redis.
func cacheResponse(ctx context.Context, redis goredis.UniversalClient, key string, resp *idempotencyResponse, ttl time.Duration) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	return redis.Set(ctx, key, data, ttl).Err()
}

// IdempotencyRequired returns a middleware that requires an idempotency key.
func IdempotencyRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.GetHeader(IdempotencyKeyHeader) == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": gin.H{
						"code":    "IDEMPOTENCY_KEY_REQUIRED",
						"message": "Idempotency-Key header is required for this request",
					},
				})
				return
			}
		}
		c.Next()
	}
}

// bodyHashKey generates a hash of the request body for additional verification.
func bodyHashKey(c *gin.Context) string {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	// Restore the body for later use
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}
