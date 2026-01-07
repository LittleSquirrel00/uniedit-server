package quota

import (
	"context"

	"github.com/google/uuid"
)

// AITaskQuotaAdapter provides AI task quota checking for the AI module.
type AITaskQuotaAdapter struct {
	checker *TaskQuotaChecker
}

// NewAITaskQuotaAdapter creates a new AI task quota adapter.
func NewAITaskQuotaAdapter(checker *TaskQuotaChecker) *AITaskQuotaAdapter {
	return &AITaskQuotaAdapter{
		checker: checker,
	}
}

// CheckChatQuota checks if user has chat token quota available.
// estimatedTokens is the estimated number of tokens for this request.
func (a *AITaskQuotaAdapter) CheckChatQuota(ctx context.Context, userID uuid.UUID, estimatedTokens int64) error {
	return a.checker.CheckChatQuota(ctx, userID, estimatedTokens)
}

// CheckImageQuota checks if user has image credit quota available.
func (a *AITaskQuotaAdapter) CheckImageQuota(ctx context.Context, userID uuid.UUID) error {
	return a.checker.CheckImageQuota(ctx, userID)
}

// CheckVideoQuota checks if user has video minutes quota available.
// estimatedMinutes is the estimated duration in minutes.
func (a *AITaskQuotaAdapter) CheckVideoQuota(ctx context.Context, userID uuid.UUID, estimatedMinutes int) error {
	return a.checker.CheckVideoQuota(ctx, userID, estimatedMinutes)
}

// CheckEmbeddingQuota checks if user has embedding token quota available.
// estimatedTokens is the estimated number of tokens for this request.
func (a *AITaskQuotaAdapter) CheckEmbeddingQuota(ctx context.Context, userID uuid.UUID, estimatedTokens int64) error {
	return a.checker.CheckEmbeddingQuota(ctx, userID, estimatedTokens)
}

// RecordChatUsage records chat token usage after request completion.
func (a *AITaskQuotaAdapter) RecordChatUsage(ctx context.Context, userID uuid.UUID, tokens int64) error {
	sub, err := a.checker.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return nil // Ignore on error
	}
	return a.checker.IncrementChatTokens(ctx, userID, sub.CurrentPeriodStart(), sub.CurrentPeriodEnd(), tokens)
}

// RecordImageUsage records image credit usage after request completion.
// count is the number of images generated.
func (a *AITaskQuotaAdapter) RecordImageUsage(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := a.checker.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return nil // Ignore on error
	}
	return a.checker.IncrementImageCredits(ctx, userID, sub.CurrentPeriodStart(), sub.CurrentPeriodEnd(), int64(count))
}

// RecordVideoUsage records video minutes usage after request completion.
// minutes is the video duration in minutes.
func (a *AITaskQuotaAdapter) RecordVideoUsage(ctx context.Context, userID uuid.UUID, minutes int) error {
	sub, err := a.checker.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return nil // Ignore on error
	}
	return a.checker.IncrementVideoMinutes(ctx, userID, sub.CurrentPeriodStart(), sub.CurrentPeriodEnd(), int64(minutes))
}

// RecordEmbeddingUsage records embedding token usage after request completion.
func (a *AITaskQuotaAdapter) RecordEmbeddingUsage(ctx context.Context, userID uuid.UUID, tokens int64) error {
	sub, err := a.checker.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return nil // Ignore on error
	}
	return a.checker.IncrementEmbeddingTokens(ctx, userID, sub.CurrentPeriodStart(), sub.CurrentPeriodEnd(), tokens)
}
