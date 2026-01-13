package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"github.com/uniedit/server/internal/utils/billingflow"
	"go.uber.org/zap"
)

func (d *Domain) CheckUsage(ctx context.Context, userID uuid.UUID, taskType string, estimatedUnits int64, estimatedCostUSD float64) error {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" || estimatedUnits < 0 || estimatedCostUSD < 0 {
		return billingflow.ErrInvalidRequest
	}

	sub, plan, err := d.getActiveSubscriptionWithPlan(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil || plan == nil {
		return billingflow.ErrQuotaExceeded
	}

	if err := d.checkDailyRequests(ctx, userID, plan); err != nil {
		return err
	}

	limit, err := usageLimitForTask(plan, taskType)
	if err != nil {
		return err
	}
	if limit == -1 || estimatedUnits == 0 {
		return nil
	}

	used, err := d.getUsedUnits(ctx, userID, sub.CurrentPeriodStart, taskType)
	if err != nil {
		d.logger.Warn("failed to get used units, assuming 0", zap.String("task_type", taskType), zap.Error(err))
		used = 0
	}

	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}

	overage := estimatedUnits - remaining
	if overage <= 0 {
		return nil
	}

	requiredCents := estimateOverageCents(estimatedUnits, estimatedCostUSD, overage)
	if requiredCents <= 0 {
		return nil
	}

	if sub.CreditsBalance < requiredCents {
		return billingflow.ErrInsufficientCredits
	}

	return nil
}

func (d *Domain) CommitUsage(ctx context.Context, userID uuid.UUID, taskType string, units int64, costUSD float64) (float64, error) {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" || units < 0 || costUSD < 0 {
		return 0, billingflow.ErrInvalidRequest
	}

	sub, plan, err := d.getActiveSubscriptionWithPlan(ctx, userID)
	if err != nil {
		return 0, err
	}
	if sub == nil || plan == nil {
		return 0, billingflow.ErrQuotaExceeded
	}

	limit, err := usageLimitForTask(plan, taskType)
	if err != nil {
		return 0, err
	}

	usedBefore, usedAfter := int64(0), int64(0)
	switch {
	case isTokenTask(taskType):
		if d.quotaCache != nil {
			newVal, err := d.quotaCache.IncrementTokens(ctx, userID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, units)
			if err == nil {
				usedAfter = newVal
				usedBefore = newVal - units
			} else {
				d.logger.Warn("failed to increment tokens in cache, using DB", zap.Error(err))
			}
		}
		if usedAfter == 0 && units > 0 {
			usedBefore, _ = d.usageDB.GetMonthlyTokensByTaskType(ctx, userID, sub.CurrentPeriodStart, taskType)
			usedAfter = usedBefore + units
		}
	default:
		if d.quotaCache != nil {
			newVal, err := d.quotaCache.IncrementMediaUnits(ctx, userID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, taskType, units)
			if err == nil {
				usedAfter = newVal
				usedBefore = newVal - units
			} else {
				d.logger.Warn("failed to increment media units in cache, using DB", zap.Error(err))
			}
		}
		if usedAfter == 0 && units > 0 {
			usedBefore, _ = d.usageDB.GetMonthlyUnitsByTaskType(ctx, userID, sub.CurrentPeriodStart, taskType)
			usedAfter = usedBefore + units
		}
	}

	overage := overageUnits(usedBefore, usedAfter, limit)
	chargeCents := estimateOverageCents(units, costUSD, overage)
	if chargeCents > 0 {
		ok, err := d.subscriptionDB.TryDeductCredits(ctx, userID, chargeCents)
		if err != nil {
			return 0, fmt.Errorf("deduct credits: %w", err)
		}
		if !ok {
			return 0, billingflow.ErrInsufficientCredits
		}
	}

	if d.quotaCache != nil {
		if _, err := d.quotaCache.IncrementRequests(ctx, userID); err != nil && !errors.Is(err, outbound.ErrCacheMiss) {
			d.logger.Warn("failed to increment requests in cache", zap.Error(err))
		}
	}

	return float64(chargeCents) / 100, nil
}

func (d *Domain) getActiveSubscriptionWithPlan(ctx context.Context, userID uuid.UUID) (*model.Subscription, *model.Plan, error) {
	sub, err := d.subscriptionDB.GetByUserIDWithPlan(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if sub == nil || !sub.IsActive() {
		return nil, nil, nil
	}
	if sub.Plan == nil {
		return nil, nil, fmt.Errorf("subscription has no plan loaded")
	}
	return sub, sub.Plan, nil
}

func (d *Domain) checkDailyRequests(ctx context.Context, userID uuid.UUID, plan *model.Plan) error {
	if plan == nil || plan.IsUnlimitedRequests() {
		return nil
	}

	var requestsToday int
	var err error
	if d.quotaCache != nil {
		requestsToday, err = d.quotaCache.GetRequestsToday(ctx, userID)
		if err != nil && !errors.Is(err, outbound.ErrCacheMiss) {
			d.logger.Warn("failed to get requests from cache, using DB", zap.Error(err))
		}
	}
	if err != nil || d.quotaCache == nil {
		requestsToday, _ = d.usageDB.GetDailyRequests(ctx, userID, time.Now().UTC())
	}

	if requestsToday >= plan.DailyRequests {
		return billingflow.ErrRateLimitExceeded
	}
	return nil
}

func usageLimitForTask(plan *model.Plan, taskType string) (int64, error) {
	if plan == nil {
		return 0, billingflow.ErrQuotaExceeded
	}

	switch taskType {
	case string(model.AITaskTypeChat):
		return plan.GetEffectiveChatTokenLimit(), nil
	case string(model.AITaskTypeEmbedding):
		if plan.MonthlyEmbeddingTokens != 0 {
			return plan.MonthlyEmbeddingTokens, nil
		}
		return plan.MonthlyTokens, nil
	case string(model.AITaskTypeImage):
		return int64(plan.MonthlyImageCredits), nil
	case string(model.AITaskTypeVideo):
		return int64(plan.MonthlyVideoMinutes), nil
	default:
		return 0, billingflow.ErrInvalidRequest
	}
}

func (d *Domain) getUsedUnits(ctx context.Context, userID uuid.UUID, periodStart time.Time, taskType string) (int64, error) {
	switch {
	case isTokenTask(taskType):
		if d.quotaCache != nil {
			v, err := d.quotaCache.GetTokensUsed(ctx, userID, periodStart)
			if err == nil {
				return v, nil
			}
			if !errors.Is(err, outbound.ErrCacheMiss) {
				return 0, err
			}
		}
		return d.usageDB.GetMonthlyTokensByTaskType(ctx, userID, periodStart, taskType)
	default:
		if d.quotaCache != nil {
			v, err := d.quotaCache.GetMediaUnitsUsed(ctx, userID, periodStart, taskType)
			if err == nil {
				return v, nil
			}
			if !errors.Is(err, outbound.ErrCacheMiss) {
				return 0, err
			}
		}
		return d.usageDB.GetMonthlyUnitsByTaskType(ctx, userID, periodStart, taskType)
	}
}

func isTokenTask(taskType string) bool {
	switch taskType {
	case string(model.AITaskTypeChat), string(model.AITaskTypeEmbedding):
		return true
	default:
		return false
	}
}

func overageUnits(usedBefore, usedAfter, limit int64) int64 {
	if limit == -1 {
		return 0
	}
	return max64(0, usedAfter-limit) - max64(0, usedBefore-limit)
}

func estimateOverageCents(totalUnits int64, totalCostUSD float64, overageUnits int64) int64 {
	if overageUnits <= 0 || totalUnits <= 0 || totalCostUSD <= 0 {
		return 0
	}
	overageCostUSD := totalCostUSD * (float64(overageUnits) / float64(totalUnits))
	return usdToCentsCeil(overageCostUSD)
}

func usdToCentsCeil(usd float64) int64 {
	if usd <= 0 {
		return 0
	}
	// Ceil with a small epsilon to avoid floating point edge cases (e.g. 1.0000000002).
	return int64(math.Ceil(usd*100 - 1e-9))
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

