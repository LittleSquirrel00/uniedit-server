package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/billing"
)

// GetUsageStatsQuery represents a query to get usage statistics.
type GetUsageStatsQuery struct {
	UserID    uuid.UUID
	StartDate time.Time
	EndDate   time.Time
}

// GetUsageStatsResult is the result of getting usage statistics.
type GetUsageStatsResult struct {
	Stats *billing.UsageStats
}

// GetUsageStatsHandler handles GetUsageStatsQuery.
type GetUsageStatsHandler struct {
	repo billing.Repository
}

// NewGetUsageStatsHandler creates a new handler.
func NewGetUsageStatsHandler(repo billing.Repository) *GetUsageStatsHandler {
	return &GetUsageStatsHandler{repo: repo}
}

// Handle executes the query.
func (h *GetUsageStatsHandler) Handle(ctx context.Context, query GetUsageStatsQuery) (*GetUsageStatsResult, error) {
	stats, err := h.repo.GetUsageStats(ctx, query.UserID, query.StartDate, query.EndDate)
	if err != nil {
		return nil, err
	}

	return &GetUsageStatsResult{Stats: stats}, nil
}
