package quota

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QuotaChecker defines the interface for checking quotas.
type QuotaChecker interface {
	CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error
}

// Checker is a middleware that checks user quota before allowing requests.
type Checker struct {
	quotaChecker QuotaChecker
	logger       *zap.Logger
}

// NewChecker creates a new quota checker middleware.
func NewChecker(quotaChecker QuotaChecker, logger *zap.Logger) *Checker {
	return &Checker{
		quotaChecker: quotaChecker,
		logger:       logger,
	}
}

// Middleware returns a Gin middleware that checks quota.
func (c *Checker) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get user ID from context
		userIDVal, exists := ctx.Get("user_id")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			ctx.Abort()
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
			ctx.Abort()
			return
		}

		// Determine task type from path or request
		taskType := determineTaskType(ctx)

		// Check quota
		if err := c.quotaChecker.CheckQuota(ctx.Request.Context(), userID, taskType); err != nil {
			if isQuotaExceeded(err) {
				ctx.JSON(http.StatusPaymentRequired, gin.H{
					"error":   "quota_exceeded",
					"message": "Your quota has been exceeded. Please upgrade your plan or wait for the quota to reset.",
				})
				ctx.Abort()
				return
			}
			c.logger.Error("quota check failed", zap.Error(err), zap.String("user_id", userID.String()))
			// On error, allow the request (graceful degradation)
		}

		ctx.Next()
	}
}

// isQuotaExceeded checks if the error indicates quota exceeded.
func isQuotaExceeded(err error) bool {
	return errors.Is(err, ErrTokenQuotaExceeded) || errors.Is(err, ErrRequestQuotaExceeded)
}

// determineTaskType determines the task type from the request.
func determineTaskType(ctx *gin.Context) string {
	path := ctx.Request.URL.Path

	// Simple path-based detection
	switch {
	case contains(path, "/chat"):
		return "chat"
	case contains(path, "/image"):
		return "image"
	case contains(path, "/video"):
		return "video"
	case contains(path, "/embedding"):
		return "embedding"
	default:
		return "chat"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
