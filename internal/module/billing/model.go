package billing

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

// BillingCycle represents the billing period.
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// Plan represents a subscription plan.
type Plan struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	Type          PlanType       `json:"type" gorm:"not null"`
	Name          string         `json:"name" gorm:"not null"`
	Description   string         `json:"description"`
	BillingCycle  BillingCycle   `json:"billing_cycle,omitempty"`
	PriceUSD      int64          `json:"price_usd"`                             // In cents
	StripePriceID string         `json:"-"`                                     // Stripe Price ID
	MonthlyTokens int64          `json:"monthly_tokens"`                        // -1 for unlimited
	DailyRequests int            `json:"daily_requests"`                        // -1 for unlimited
	MaxAPIKeys    int            `json:"max_api_keys"`
	Features      pq.StringArray `json:"features" gorm:"type:text[]"`
	Active        bool           `json:"active" gorm:"default:true"`
	DisplayOrder  int            `json:"display_order" gorm:"default:0"`

	// Task-specific quotas
	MonthlyChatTokens      int64 `json:"monthly_chat_tokens" gorm:"default:0"`      // -1=unlimited, 0=use MonthlyTokens
	MonthlyImageCredits    int   `json:"monthly_image_credits" gorm:"default:0"`    // -1=unlimited
	MonthlyVideoMinutes    int   `json:"monthly_video_minutes" gorm:"default:0"`    // -1=unlimited
	MonthlyEmbeddingTokens int64 `json:"monthly_embedding_tokens" gorm:"default:0"` // -1=unlimited

	// Storage quotas
	GitStorageMB int64 `json:"git_storage_mb" gorm:"default:-1"` // -1=unlimited
	LFSStorageMB int64 `json:"lfs_storage_mb" gorm:"default:-1"` // -1=unlimited

	// Team quota
	MaxTeamMembers int `json:"max_team_members" gorm:"default:5"` // -1=unlimited

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

// IsUnlimitedChatTokens returns true if chat tokens are unlimited.
func (p *Plan) IsUnlimitedChatTokens() bool {
	return p.MonthlyChatTokens == -1
}

// IsUnlimitedImageCredits returns true if image credits are unlimited.
func (p *Plan) IsUnlimitedImageCredits() bool {
	return p.MonthlyImageCredits == -1
}

// IsUnlimitedVideoMinutes returns true if video minutes are unlimited.
func (p *Plan) IsUnlimitedVideoMinutes() bool {
	return p.MonthlyVideoMinutes == -1
}

// IsUnlimitedEmbeddingTokens returns true if embedding tokens are unlimited.
func (p *Plan) IsUnlimitedEmbeddingTokens() bool {
	return p.MonthlyEmbeddingTokens == -1
}

// IsUnlimitedGitStorage returns true if git storage is unlimited.
func (p *Plan) IsUnlimitedGitStorage() bool {
	return p.GitStorageMB == -1
}

// IsUnlimitedLFSStorage returns true if LFS storage is unlimited.
func (p *Plan) IsUnlimitedLFSStorage() bool {
	return p.LFSStorageMB == -1
}

// IsUnlimitedTeamMembers returns true if team members are unlimited.
func (p *Plan) IsUnlimitedTeamMembers() bool {
	return p.MaxTeamMembers == -1
}

// GetEffectiveChatTokenLimit returns the effective chat token limit.
// If MonthlyChatTokens is 0, falls back to MonthlyTokens for backward compatibility.
func (p *Plan) GetEffectiveChatTokenLimit() int64 {
	if p.MonthlyChatTokens == 0 {
		return p.MonthlyTokens
	}
	return p.MonthlyChatTokens
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

// Subscription represents a user's subscription to a plan.
type Subscription struct {
	ID                   uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID               uuid.UUID          `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	PlanID               string             `json:"plan_id" gorm:"not null"`
	Status               SubscriptionStatus `json:"status" gorm:"not null;default:active"`
	StripeCustomerID     string             `json:"-"`
	StripeSubscriptionID string             `json:"-"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end" gorm:"default:false"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	CreditsBalance       int64              `json:"credits_balance" gorm:"default:0"` // In cents
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`

	// Relations
	Plan *Plan `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
}

// TableName returns the database table name.
func (Subscription) TableName() string {
	return "subscriptions"
}

// IsActive returns true if the subscription is active or trialing.
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsCanceled returns true if the subscription is canceled.
func (s *Subscription) IsCanceled() bool {
	return s.Status == SubscriptionStatusCanceled
}

// UsageRecord represents a single AI usage event.
type UsageRecord struct {
	ID           int64      `gorm:"primaryKey;autoIncrement"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null"`
	APIKeyID     *uuid.UUID `gorm:"type:uuid"` // System API key used (nil for JWT auth)
	Timestamp    time.Time  `gorm:"not null"`
	RequestID    string     `gorm:"not null"`
	TaskType     string     `gorm:"not null"` // chat, image, video, embedding
	ProviderID   uuid.UUID  `gorm:"type:uuid;not null"`
	ModelID      string     `gorm:"not null"`
	InputTokens  int        `gorm:"not null;default:0"`
	OutputTokens int        `gorm:"not null;default:0"`
	TotalTokens  int        `gorm:"not null;default:0"`
	CostUSD      float64    `gorm:"type:decimal(10,6);not null"`
	LatencyMs    int        `gorm:"not null"`
	Success      bool       `gorm:"not null"`

	// Cache statistics
	CacheHit bool `gorm:"not null;default:false"` // Whether cache was hit
}

// TableName returns the database table name.
func (UsageRecord) TableName() string {
	return "usage_records"
}
