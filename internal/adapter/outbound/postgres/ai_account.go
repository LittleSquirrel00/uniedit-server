package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// aiProviderAccountAdapter implements outbound.AIProviderAccountDatabasePort.
type aiProviderAccountAdapter struct {
	db *gorm.DB
}

// NewAIProviderAccountAdapter creates a new AI provider account database adapter.
func NewAIProviderAccountAdapter(db *gorm.DB) outbound.AIProviderAccountDatabasePort {
	return &aiProviderAccountAdapter{db: db}
}

func (a *aiProviderAccountAdapter) Create(ctx context.Context, account *model.AIProviderAccount) error {
	return a.db.WithContext(ctx).Create(account).Error
}

func (a *aiProviderAccountAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.AIProviderAccount, error) {
	var account model.AIProviderAccount
	err := a.db.WithContext(ctx).First(&account, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *aiProviderAccountAdapter) FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	var accounts []*model.AIProviderAccount
	err := a.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	return accounts, err
}

func (a *aiProviderAccountAdapter) FindActiveByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	var accounts []*model.AIProviderAccount
	err := a.db.WithContext(ctx).
		Where("provider_id = ? AND is_active = ?", providerID, true).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	return accounts, err
}

func (a *aiProviderAccountAdapter) FindAvailableByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	var accounts []*model.AIProviderAccount
	err := a.db.WithContext(ctx).
		Where("provider_id = ? AND is_active = ? AND health_status IN ?",
			providerID, true, []string{string(model.AIHealthStatusHealthy), string(model.AIHealthStatusDegraded)}).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	return accounts, err
}

func (a *aiProviderAccountAdapter) Update(ctx context.Context, account *model.AIProviderAccount) error {
	return a.db.WithContext(ctx).Save(account).Error
}

func (a *aiProviderAccountAdapter) UpdateHealth(ctx context.Context, id uuid.UUID, status model.AIHealthStatus, consecutiveFailures int) error {
	now := time.Now()
	updates := map[string]interface{}{
		"health_status":        status,
		"consecutive_failures": consecutiveFailures,
		"last_health_check":    now,
		"updated_at":           now,
	}

	if status != model.AIHealthStatusHealthy {
		updates["last_failure_at"] = now
	}

	return a.db.WithContext(ctx).
		Model(&model.AIProviderAccount{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (a *aiProviderAccountAdapter) IncrementUsage(ctx context.Context, id uuid.UUID, requests, tokens int64, cost float64) error {
	return a.db.WithContext(ctx).
		Model(&model.AIProviderAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_requests":       gorm.Expr("total_requests + ?", requests),
			"total_tokens":         gorm.Expr("total_tokens + ?", tokens),
			"total_cost_usd":       gorm.Expr("total_cost_usd + ?", cost),
			"consecutive_failures": 0, // Reset on success
			"health_status":        model.AIHealthStatusHealthy,
			"updated_at":           time.Now(),
		}).Error
}

func (a *aiProviderAccountAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.AIProviderAccount{}, "id = ?", id).Error
}

func (a *aiProviderAccountAdapter) DeleteByProvider(ctx context.Context, providerID uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.AIProviderAccount{}, "provider_id = ?", providerID).Error
}

// Compile-time check
var _ outbound.AIProviderAccountDatabasePort = (*aiProviderAccountAdapter)(nil)

// aiAccountUsageStatsAdapter implements outbound.AIAccountUsageStatsDatabasePort.
type aiAccountUsageStatsAdapter struct {
	db *gorm.DB
}

// NewAIAccountUsageStatsAdapter creates a new account usage stats database adapter.
func NewAIAccountUsageStatsAdapter(db *gorm.DB) outbound.AIAccountUsageStatsDatabasePort {
	return &aiAccountUsageStatsAdapter{db: db}
}

func (a *aiAccountUsageStatsAdapter) RecordUsage(ctx context.Context, accountID uuid.UUID, requests, tokens int64, cost float64) error {
	today := time.Now().Truncate(24 * time.Hour)

	// Upsert daily stats
	return a.db.WithContext(ctx).Exec(`
		INSERT INTO account_usage_stats (account_id, date, requests_count, tokens_count, cost_usd, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())
		ON CONFLICT (account_id, date)
		DO UPDATE SET
			requests_count = account_usage_stats.requests_count + EXCLUDED.requests_count,
			tokens_count = account_usage_stats.tokens_count + EXCLUDED.tokens_count,
			cost_usd = account_usage_stats.cost_usd + EXCLUDED.cost_usd
	`, accountID, today, requests, tokens, cost).Error
}

func (a *aiAccountUsageStatsAdapter) FindByAccount(ctx context.Context, accountID uuid.UUID, days int) ([]*model.AIAccountUsageStats, error) {
	var stats []*model.AIAccountUsageStats
	err := a.db.WithContext(ctx).
		Where("account_id = ? AND date >= ?", accountID, time.Now().AddDate(0, 0, -days)).
		Order("date DESC").
		Find(&stats).Error
	return stats, err
}

func (a *aiAccountUsageStatsAdapter) FindByAccountAndDateRange(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.AIAccountUsageStats, error) {
	var stats []*model.AIAccountUsageStats
	err := a.db.WithContext(ctx).
		Where("account_id = ? AND date >= ? AND date <= ?", accountID, start, end).
		Order("date DESC").
		Find(&stats).Error
	return stats, err
}

// Compile-time check
var _ outbound.AIAccountUsageStatsDatabasePort = (*aiAccountUsageStatsAdapter)(nil)
