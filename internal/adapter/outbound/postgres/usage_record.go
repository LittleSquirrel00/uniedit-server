package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// usageRecordAdapter implements outbound.UsageRecordDatabasePort.
type usageRecordAdapter struct {
	db *gorm.DB
}

// NewUsageRecordAdapter creates a new usage record database adapter.
func NewUsageRecordAdapter(db *gorm.DB) outbound.UsageRecordDatabasePort {
	return &usageRecordAdapter{db: db}
}

func (a *usageRecordAdapter) Create(ctx context.Context, record *model.UsageRecord) error {
	return a.db.WithContext(ctx).Create(record).Error
}

func (a *usageRecordAdapter) GetStats(ctx context.Context, userID uuid.UUID, start, end time.Time) (*model.UsageStats, error) {
	var totalStats struct {
		TotalTokens   int64
		TotalRequests int
		TotalCostUSD  float64
	}

	// Get totals
	err := a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Select("COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND success = true AND timestamp >= ? AND timestamp < ?", userID, start, end).
		Scan(&totalStats).Error
	if err != nil {
		return nil, err
	}

	// Get by model
	var modelStats []struct {
		ModelID       string
		TotalTokens   int64
		TotalRequests int
		TotalCostUSD  float64
	}
	err = a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Select("model_id, COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND success = true AND timestamp >= ? AND timestamp < ?", userID, start, end).
		Group("model_id").
		Scan(&modelStats).Error
	if err != nil {
		return nil, err
	}

	byModel := make(map[string]*model.ModelUsage)
	for _, ms := range modelStats {
		byModel[ms.ModelID] = &model.ModelUsage{
			ModelID:       ms.ModelID,
			TotalTokens:   ms.TotalTokens,
			TotalRequests: ms.TotalRequests,
			TotalCostUSD:  ms.TotalCostUSD,
		}
	}

	// Get by day
	var dailyStats []struct {
		Date          string
		TotalTokens   int64
		TotalRequests int
		TotalCostUSD  float64
	}
	err = a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Select("DATE(timestamp) as date, COALESCE(SUM(total_tokens), 0) as total_tokens, COUNT(*) as total_requests, COALESCE(SUM(cost_usd), 0) as total_cost_usd").
		Where("user_id = ? AND success = true AND timestamp >= ? AND timestamp < ?", userID, start, end).
		Group("DATE(timestamp)").
		Order("date ASC").
		Scan(&dailyStats).Error
	if err != nil {
		return nil, err
	}

	byDay := make([]*model.DailyUsage, len(dailyStats))
	for i, ds := range dailyStats {
		byDay[i] = &model.DailyUsage{
			Date:          ds.Date,
			TotalTokens:   ds.TotalTokens,
			TotalRequests: ds.TotalRequests,
			TotalCostUSD:  ds.TotalCostUSD,
		}
	}

	return &model.UsageStats{
		TotalTokens:   totalStats.TotalTokens,
		TotalRequests: totalStats.TotalRequests,
		TotalCostUSD:  totalStats.TotalCostUSD,
		ByModel:       byModel,
		ByDay:         byDay,
	}, nil
}

func (a *usageRecordAdapter) GetMonthlyTokens(ctx context.Context, userID uuid.UUID, start time.Time) (int64, error) {
	var total int64
	err := a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Select("COALESCE(SUM(total_tokens), 0)").
		Where("user_id = ? AND success = true AND timestamp >= ?", userID, start).
		Scan(&total).Error
	return total, err
}

func (a *usageRecordAdapter) GetMonthlyTokensByTaskType(ctx context.Context, userID uuid.UUID, start time.Time, taskType string) (int64, error) {
	var total int64
	err := a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Select("COALESCE(SUM(total_tokens), 0)").
		Where("user_id = ? AND success = true AND task_type = ? AND timestamp >= ?", userID, taskType, start).
		Scan(&total).Error
	return total, err
}

func (a *usageRecordAdapter) GetDailyRequests(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var count int64
	err := a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Where("user_id = ? AND success = true AND timestamp >= ? AND timestamp < ?", userID, startOfDay, endOfDay).
		Count(&count).Error
	return int(count), err
}

func (a *usageRecordAdapter) GetMonthlyUnitsByTaskType(ctx context.Context, userID uuid.UUID, start time.Time, taskType string) (int64, error) {
	var total int64
	err := a.db.WithContext(ctx).
		Model(&model.UsageRecord{}).
		Select("COALESCE(SUM(input_tokens), 0)").
		Where("user_id = ? AND success = true AND task_type = ? AND timestamp >= ?", userID, taskType, start).
		Scan(&total).Error
	return total, err
}

// Compile-time check
var _ outbound.UsageRecordDatabasePort = (*usageRecordAdapter)(nil)
