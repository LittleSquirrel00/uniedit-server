package pool

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Common errors.
var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrNoAvailableAccount = errors.New("no available account")
	ErrDuplicateAccount   = errors.New("account with this name already exists")
)

// Repository defines the interface for provider account persistence.
type Repository interface {
	// Create adds a new account to the pool.
	Create(ctx context.Context, account *ProviderAccount) error

	// GetByID returns an account by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*ProviderAccount, error)

	// GetActiveByProvider returns all active accounts for a provider.
	GetActiveByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderAccount, error)

	// GetAllByProvider returns all accounts for a provider.
	GetAllByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderAccount, error)

	// Update updates an account.
	Update(ctx context.Context, account *ProviderAccount) error

	// Delete removes an account from the pool.
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateHealthStatus updates the health status of an account.
	UpdateHealthStatus(ctx context.Context, id uuid.UUID, status HealthStatus, failures int) error

	// RecordSuccess records a successful request.
	RecordSuccess(ctx context.Context, id uuid.UUID, tokens int, costUSD float64) error

	// RecordFailure records a failed request.
	RecordFailure(ctx context.Context, id uuid.UUID) error

	// GetStats returns usage statistics for an account.
	GetStats(ctx context.Context, id uuid.UUID, days int) (*AccountStats, error)

	// RecordDailyUsage records daily usage statistics.
	RecordDailyUsage(ctx context.Context, accountID uuid.UUID, tokens int, costUSD float64) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, account *ProviderAccount) error {
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateAccount
		}
		return err
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*ProviderAccount, error) {
	var account ProviderAccount
	if err := r.db.WithContext(ctx).First(&account, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (r *repository) GetActiveByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderAccount, error) {
	var accounts []*ProviderAccount
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND is_active = true", providerID).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *repository) GetAllByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderAccount, error) {
	var accounts []*ProviderAccount
	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("priority DESC, created_at ASC").
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *repository) Update(ctx context.Context, account *ProviderAccount) error {
	result := r.db.WithContext(ctx).Save(account)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&ProviderAccount{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *repository) UpdateHealthStatus(ctx context.Context, id uuid.UUID, status HealthStatus, failures int) error {
	now := time.Now()
	updates := map[string]any{
		"health_status":        status,
		"consecutive_failures": failures,
		"last_health_check":    now,
		"updated_at":           now,
	}
	if status != HealthStatusHealthy && failures > 0 {
		updates["last_failure_at"] = now
	}
	result := r.db.WithContext(ctx).Model(&ProviderAccount{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *repository) RecordSuccess(ctx context.Context, id uuid.UUID, tokens int, costUSD float64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&ProviderAccount{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"total_requests":       gorm.Expr("total_requests + 1"),
			"total_tokens":         gorm.Expr("total_tokens + ?", tokens),
			"total_cost_usd":       gorm.Expr("total_cost_usd + ?", costUSD),
			"consecutive_failures": 0,
			"last_health_check":    now,
			"updated_at":           now,
		})
	return result.Error
}

func (r *repository) RecordFailure(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&ProviderAccount{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"consecutive_failures": gorm.Expr("consecutive_failures + 1"),
			"last_failure_at":      now,
			"last_health_check":    now,
			"updated_at":           now,
		})
	return result.Error
}

func (r *repository) GetStats(ctx context.Context, id uuid.UUID, days int) (*AccountStats, error) {
	account, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	stats := &AccountStats{
		AccountID:           account.ID,
		TotalRequests:       account.TotalRequests,
		TotalTokens:         account.TotalTokens,
		TotalCostUSD:        account.TotalCostUSD,
		HealthStatus:        account.HealthStatus,
		ConsecutiveFailures: account.ConsecutiveFailures,
		LastHealthCheck:     account.LastHealthCheck,
	}

	// Get daily stats
	if days > 0 {
		startDate := time.Now().AddDate(0, 0, -days)
		var dailyStats []AccountUsageStats
		err := r.db.WithContext(ctx).
			Where("account_id = ? AND date >= ?", id, startDate).
			Order("date DESC").
			Find(&dailyStats).Error
		if err != nil {
			return nil, err
		}
		stats.DailyStats = dailyStats
	}

	return stats, nil
}

func (r *repository) RecordDailyUsage(ctx context.Context, accountID uuid.UUID, tokens int, costUSD float64) error {
	today := time.Now().Truncate(24 * time.Hour)

	// Upsert daily usage
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO account_usage_stats (account_id, date, requests_count, tokens_count, cost_usd, created_at)
		VALUES (?, ?, 1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (account_id, date) DO UPDATE SET
			requests_count = account_usage_stats.requests_count + 1,
			tokens_count = account_usage_stats.tokens_count + EXCLUDED.tokens_count,
			cost_usd = account_usage_stats.cost_usd + EXCLUDED.cost_usd
	`, accountID, today, tokens, costUSD)

	return result.Error
}
