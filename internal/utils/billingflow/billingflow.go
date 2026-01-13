package billingflow

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest      = errors.New("invalid request")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrInsufficientCredits = errors.New("insufficient credits")
)

// UsageBiller checks and commits hybrid billing for a task type.
// The caller provides estimated usage for pre-check and actual usage for commit.
type UsageBiller interface {
	CheckUsage(ctx context.Context, userID uuid.UUID, taskType string, estimatedUnits int64, estimatedCostUSD float64) error
	CommitUsage(ctx context.Context, userID uuid.UUID, taskType string, units int64, costUSD float64) (chargedCostUSD float64, err error)
}

