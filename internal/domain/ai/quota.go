package ai

import (
	"context"

	"github.com/google/uuid"
)

// QuotaChecker is an optional dependency used by the AI domain to enforce user-level quotas
// (tokens / request counts) before making upstream calls, and to consume quota after success.
//
// It intentionally lives in the ai package (consumer side) to avoid direct dependency on billing domain types.
type QuotaChecker interface {
	CheckQuota(ctx context.Context, userID uuid.UUID, taskType string) error
	ConsumeQuota(ctx context.Context, userID uuid.UUID, tokens int) error
}
