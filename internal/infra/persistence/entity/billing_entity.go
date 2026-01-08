package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uniedit/server/internal/domain/billing"
)

// PlanEntity is the GORM model for plans table.
type PlanEntity struct {
	ID            string              `gorm:"primaryKey"`
	Type          billing.PlanType    `gorm:"not null"`
	Name          string              `gorm:"not null"`
	Description   string
	BillingCycle  billing.BillingCycle
	PriceUSD      int64
	StripePriceID string
	MonthlyTokens int64
	DailyRequests int
	MaxAPIKeys    int
	Features      pq.StringArray `gorm:"type:text[]"`
	Active        bool           `gorm:"default:true"`
	DisplayOrder  int            `gorm:"default:0"`

	MonthlyChatTokens      int64 `gorm:"default:0"`
	MonthlyImageCredits    int   `gorm:"default:0"`
	MonthlyVideoMinutes    int   `gorm:"default:0"`
	MonthlyEmbeddingTokens int64 `gorm:"default:0"`

	GitStorageMB   int64 `gorm:"default:-1"`
	LFSStorageMB   int64 `gorm:"default:-1"`
	MaxTeamMembers int   `gorm:"default:5"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the database table name.
func (PlanEntity) TableName() string {
	return "plans"
}

// ToDomain converts the entity to a domain Plan.
func (e *PlanEntity) ToDomain() *billing.Plan {
	return billing.RestorePlan(
		e.ID,
		e.Type,
		e.Name,
		e.Description,
		e.BillingCycle,
		e.PriceUSD,
		e.StripePriceID,
		e.MonthlyTokens,
		e.DailyRequests,
		e.MaxAPIKeys,
		[]string(e.Features),
		e.Active,
		e.DisplayOrder,
		e.MonthlyChatTokens,
		e.MonthlyImageCredits,
		e.MonthlyVideoMinutes,
		e.MonthlyEmbeddingTokens,
		e.GitStorageMB,
		e.LFSStorageMB,
		e.MaxTeamMembers,
	)
}

// SubscriptionEntity is the GORM model for subscriptions table.
type SubscriptionEntity struct {
	ID                   uuid.UUID                   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID               uuid.UUID                   `gorm:"type:uuid;uniqueIndex;not null"`
	PlanID               string                      `gorm:"not null"`
	Status               billing.SubscriptionStatus  `gorm:"not null;default:active"`
	StripeCustomerID     string
	StripeSubscriptionID string
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CancelAtPeriodEnd    bool       `gorm:"default:false"`
	CanceledAt           *time.Time
	CreditsBalance       int64      `gorm:"default:0"`
	CreatedAt            time.Time
	UpdatedAt            time.Time

	// Relations
	Plan *PlanEntity `gorm:"foreignKey:PlanID"`
}

// TableName returns the database table name.
func (SubscriptionEntity) TableName() string {
	return "subscriptions"
}

// ToDomain converts the entity to a domain Subscription.
func (e *SubscriptionEntity) ToDomain() *billing.Subscription {
	var plan *billing.Plan
	if e.Plan != nil {
		plan = e.Plan.ToDomain()
	}

	return billing.RestoreSubscription(
		e.ID,
		e.UserID,
		e.PlanID,
		e.Status,
		e.StripeCustomerID,
		e.StripeSubscriptionID,
		e.CurrentPeriodStart,
		e.CurrentPeriodEnd,
		e.CancelAtPeriodEnd,
		e.CanceledAt,
		e.CreditsBalance,
		e.CreatedAt,
		e.UpdatedAt,
		plan,
	)
}

// FromDomainSubscription converts a domain Subscription to an entity.
func FromDomainSubscription(s *billing.Subscription) *SubscriptionEntity {
	return &SubscriptionEntity{
		ID:                   s.ID(),
		UserID:               s.UserID(),
		PlanID:               s.PlanID(),
		Status:               s.Status(),
		StripeCustomerID:     s.StripeCustomerID(),
		StripeSubscriptionID: s.StripeSubscriptionID(),
		CurrentPeriodStart:   s.CurrentPeriodStart(),
		CurrentPeriodEnd:     s.CurrentPeriodEnd(),
		CancelAtPeriodEnd:    s.CancelAtPeriodEnd(),
		CanceledAt:           s.CanceledAt(),
		CreditsBalance:       s.CreditsBalance(),
		CreatedAt:            s.CreatedAt(),
		UpdatedAt:            s.UpdatedAt(),
	}
}

// UsageRecordEntity is the GORM model for usage_records table.
type UsageRecordEntity struct {
	ID           int64      `gorm:"primaryKey;autoIncrement"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null"`
	APIKeyID     *uuid.UUID `gorm:"type:uuid"`
	Timestamp    time.Time  `gorm:"not null"`
	RequestID    string     `gorm:"not null"`
	TaskType     string     `gorm:"not null"`
	ProviderID   uuid.UUID  `gorm:"type:uuid;not null"`
	ModelID      string     `gorm:"not null"`
	InputTokens  int        `gorm:"not null;default:0"`
	OutputTokens int        `gorm:"not null;default:0"`
	TotalTokens  int        `gorm:"not null;default:0"`
	CostUSD      float64    `gorm:"type:decimal(10,6);not null"`
	LatencyMs    int        `gorm:"not null"`
	Success      bool       `gorm:"not null"`
	CacheHit     bool       `gorm:"not null;default:false"`
}

// TableName returns the database table name.
func (UsageRecordEntity) TableName() string {
	return "usage_records"
}

// ToDomain converts the entity to a domain UsageRecord.
func (e *UsageRecordEntity) ToDomain() *billing.UsageRecord {
	return billing.RestoreUsageRecord(
		e.ID,
		e.UserID,
		e.APIKeyID,
		e.Timestamp,
		e.RequestID,
		e.TaskType,
		e.ProviderID,
		e.ModelID,
		e.InputTokens,
		e.OutputTokens,
		e.TotalTokens,
		e.CostUSD,
		e.LatencyMs,
		e.Success,
		e.CacheHit,
	)
}

// FromDomainUsageRecord converts a domain UsageRecord to an entity.
func FromDomainUsageRecord(r *billing.UsageRecord) *UsageRecordEntity {
	return &UsageRecordEntity{
		ID:           r.ID(),
		UserID:       r.UserID(),
		APIKeyID:     r.APIKeyID(),
		Timestamp:    r.Timestamp(),
		RequestID:    r.RequestID(),
		TaskType:     r.TaskType(),
		ProviderID:   r.ProviderID(),
		ModelID:      r.ModelID(),
		InputTokens:  r.InputTokens(),
		OutputTokens: r.OutputTokens(),
		TotalTokens:  r.TotalTokens(),
		CostUSD:      r.CostUSD(),
		LatencyMs:    r.LatencyMs(),
		Success:      r.Success(),
		CacheHit:     r.CacheHit(),
	}
}
