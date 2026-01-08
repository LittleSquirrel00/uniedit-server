package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PlanType represents the type of subscription plan.
type PlanType string

const (
	PlanTypeFree       PlanType = "free"
	PlanTypePro        PlanType = "pro"
	PlanTypeTeam       PlanType = "team"
	PlanTypeEnterprise PlanType = "enterprise"
)

// String returns the string representation of the plan type.
func (p PlanType) String() string {
	return string(p)
}

// IsValid checks if the plan type is valid.
func (p PlanType) IsValid() bool {
	switch p {
	case PlanTypeFree, PlanTypePro, PlanTypeTeam, PlanTypeEnterprise:
		return true
	}
	return false
}

// BillingCycle represents the billing period.
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// String returns the string representation of the billing cycle.
func (b BillingCycle) String() string {
	return string(b)
}

// IsValid checks if the billing cycle is valid.
func (b BillingCycle) IsValid() bool {
	switch b {
	case BillingCycleMonthly, BillingCycleYearly:
		return true
	}
	return false
}

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing   SubscriptionStatus = "trialing"
	SubscriptionStatusActive     SubscriptionStatus = "active"
	SubscriptionStatusPastDue    SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled   SubscriptionStatus = "canceled"
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"
)

// String returns the string representation of the status.
func (s SubscriptionStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid.
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case SubscriptionStatusTrialing, SubscriptionStatusActive, SubscriptionStatusPastDue, SubscriptionStatusCanceled, SubscriptionStatusIncomplete:
		return true
	}
	return false
}

// IsActive returns true if the subscription is active or trialing.
func (s SubscriptionStatus) IsActive() bool {
	return s == SubscriptionStatusActive || s == SubscriptionStatusTrialing
}

// Plan represents a subscription plan.
type Plan struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	Type          PlanType       `json:"type" gorm:"not null"`
	Name          string         `json:"name" gorm:"not null"`
	Description   string         `json:"description"`
	BillingCycle  BillingCycle   `json:"billing_cycle"`
	PriceUSD      int64          `json:"price_usd"`
	StripePriceID string         `json:"stripe_price_id,omitempty"`
	Features      pq.StringArray `json:"features" gorm:"type:text[]"`
	Active        bool           `json:"active" gorm:"default:true"`
	DisplayOrder  int            `json:"display_order" gorm:"default:0"`

	// Token quotas
	MonthlyTokens          int64 `json:"monthly_tokens"`
	DailyRequests          int   `json:"daily_requests"`
	MaxAPIKeys             int   `json:"max_api_keys"`
	MonthlyChatTokens      int64 `json:"monthly_chat_tokens" gorm:"default:0"`
	MonthlyImageCredits    int   `json:"monthly_image_credits" gorm:"default:0"`
	MonthlyVideoMinutes    int   `json:"monthly_video_minutes" gorm:"default:0"`
	MonthlyEmbeddingTokens int64 `json:"monthly_embedding_tokens" gorm:"default:0"`

	// Storage quotas
	GitStorageMB int64 `json:"git_storage_mb" gorm:"default:-1"`
	LFSStorageMB int64 `json:"lfs_storage_mb" gorm:"default:-1"`

	// Team quota
	MaxTeamMembers int `json:"max_team_members" gorm:"default:5"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the database table name.
func (Plan) TableName() string {
	return "plans"
}

// IsUnlimitedTokens returns true if the plan has unlimited tokens.
func (p *Plan) IsUnlimitedTokens() bool {
	return p.MonthlyTokens == -1
}

// IsUnlimitedRequests returns true if the plan has unlimited daily requests.
func (p *Plan) IsUnlimitedRequests() bool {
	return p.DailyRequests == -1
}

// GetEffectiveChatTokenLimit returns the effective chat token limit.
func (p *Plan) GetEffectiveChatTokenLimit() int64 {
	if p.MonthlyChatTokens == 0 {
		return p.MonthlyTokens
	}
	return p.MonthlyChatTokens
}

// PlanResponse represents plan information for API responses.
type PlanResponse struct {
	ID                     string   `json:"id"`
	Type                   string   `json:"type"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	BillingCycle           string   `json:"billing_cycle,omitempty"`
	PriceUSD               int64    `json:"price_usd"`
	MonthlyTokens          int64    `json:"monthly_tokens"`
	DailyRequests          int      `json:"daily_requests"`
	MaxAPIKeys             int      `json:"max_api_keys"`
	Features               []string `json:"features"`
	MonthlyChatTokens      int64    `json:"monthly_chat_tokens"`
	MonthlyImageCredits    int      `json:"monthly_image_credits"`
	MonthlyVideoMinutes    int      `json:"monthly_video_minutes"`
	MonthlyEmbeddingTokens int64    `json:"monthly_embedding_tokens"`
	GitStorageMB           int64    `json:"git_storage_mb"`
	LFSStorageMB           int64    `json:"lfs_storage_mb"`
	MaxTeamMembers         int      `json:"max_team_members"`
}

// ToResponse converts Plan to PlanResponse.
func (p *Plan) ToResponse() *PlanResponse {
	return &PlanResponse{
		ID:                     p.ID,
		Type:                   string(p.Type),
		Name:                   p.Name,
		Description:            p.Description,
		BillingCycle:           string(p.BillingCycle),
		PriceUSD:               p.PriceUSD,
		MonthlyTokens:          p.MonthlyTokens,
		DailyRequests:          p.DailyRequests,
		MaxAPIKeys:             p.MaxAPIKeys,
		Features:               []string(p.Features),
		MonthlyChatTokens:      p.MonthlyChatTokens,
		MonthlyImageCredits:    p.MonthlyImageCredits,
		MonthlyVideoMinutes:    p.MonthlyVideoMinutes,
		MonthlyEmbeddingTokens: p.MonthlyEmbeddingTokens,
		GitStorageMB:           p.GitStorageMB,
		LFSStorageMB:           p.LFSStorageMB,
		MaxTeamMembers:         p.MaxTeamMembers,
	}
}

// Subscription represents a user subscription.
type Subscription struct {
	ID                   uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID               uuid.UUID          `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	PlanID               string             `json:"plan_id" gorm:"not null"`
	Status               SubscriptionStatus `json:"status" gorm:"not null;default:active"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end" gorm:"default:false"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	CreditsBalance       int64              `json:"credits_balance" gorm:"default:0"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`

	// Relations
	Plan *Plan `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
}

// TableName returns the database table name.
func (Subscription) TableName() string {
	return "subscriptions"
}

// IsActive returns true if the subscription is active.
func (s *Subscription) IsActive() bool {
	return s.Status.IsActive()
}

// IsCanceled returns true if the subscription is canceled.
func (s *Subscription) IsCanceled() bool {
	return s.Status == SubscriptionStatusCanceled
}

// HasSufficientCredits checks if the subscription has enough credits.
func (s *Subscription) HasSufficientCredits(amount int64) bool {
	return s.CreditsBalance >= amount
}

// SubscriptionResponse represents subscription information for API responses.
type SubscriptionResponse struct {
	ID                 uuid.UUID     `json:"id"`
	PlanID             string        `json:"plan_id"`
	Status             string        `json:"status"`
	CurrentPeriodStart time.Time     `json:"current_period_start"`
	CurrentPeriodEnd   time.Time     `json:"current_period_end"`
	CancelAtPeriodEnd  bool          `json:"cancel_at_period_end"`
	CanceledAt         *time.Time    `json:"canceled_at,omitempty"`
	CreditsBalance     int64         `json:"credits_balance"`
	CreatedAt          time.Time     `json:"created_at"`
	Plan               *PlanResponse `json:"plan,omitempty"`
}

// ToResponse converts Subscription to SubscriptionResponse.
func (s *Subscription) ToResponse() *SubscriptionResponse {
	resp := &SubscriptionResponse{
		ID:                 s.ID,
		PlanID:             s.PlanID,
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		CanceledAt:         s.CanceledAt,
		CreditsBalance:     s.CreditsBalance,
		CreatedAt:          s.CreatedAt,
	}
	if s.Plan != nil {
		resp.Plan = s.Plan.ToResponse()
	}
	return resp
}

// UsageRecord represents a single AI usage event.
type UsageRecord struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	APIKeyID     *uuid.UUID `json:"api_key_id,omitempty" gorm:"type:uuid"`
	Timestamp    time.Time  `json:"timestamp" gorm:"not null;index"`
	RequestID    string     `json:"request_id" gorm:"not null"`
	TaskType     string     `json:"task_type" gorm:"not null"`
	ProviderID   uuid.UUID  `json:"provider_id" gorm:"type:uuid;not null"`
	ModelID      string     `json:"model_id" gorm:"not null"`
	InputTokens  int        `json:"input_tokens" gorm:"not null;default:0"`
	OutputTokens int        `json:"output_tokens" gorm:"not null;default:0"`
	TotalTokens  int        `json:"total_tokens" gorm:"not null;default:0"`
	CostUSD      float64    `json:"cost_usd" gorm:"type:decimal(10,6);not null"`
	LatencyMs    int        `json:"latency_ms" gorm:"not null"`
	Success      bool       `json:"success" gorm:"not null"`
	CacheHit     bool       `json:"cache_hit" gorm:"not null;default:false"`
}

// TableName returns the database table name.
func (UsageRecord) TableName() string {
	return "usage_records"
}

// QuotaStatus represents the current quota usage status.
type QuotaStatus struct {
	Plan            string    `json:"plan"`
	TokensUsed      int64     `json:"tokens_used"`
	TokensLimit     int64     `json:"tokens_limit"`
	TokensRemaining int64     `json:"tokens_remaining"`
	RequestsToday   int       `json:"requests_today"`
	RequestsLimit   int       `json:"requests_limit"`
	ResetAt         time.Time `json:"reset_at"`
}

// UsageStats represents aggregated usage statistics.
type UsageStats struct {
	TotalTokens   int64                  `json:"total_tokens"`
	TotalRequests int                    `json:"total_requests"`
	TotalCostUSD  float64                `json:"total_cost_usd"`
	ByModel       map[string]*ModelUsage `json:"by_model"`
	ByDay         []*DailyUsage          `json:"by_day"`
}

// ModelUsage represents usage by model.
type ModelUsage struct {
	ModelID       string  `json:"model_id"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalRequests int     `json:"total_requests"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
}

// DailyUsage represents usage for a single day.
type DailyUsage struct {
	Date          string  `json:"date"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalRequests int     `json:"total_requests"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
}
