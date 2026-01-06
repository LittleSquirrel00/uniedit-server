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
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
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
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	Timestamp    time.Time `gorm:"not null"`
	RequestID    string    `gorm:"not null"`
	TaskType     string    `gorm:"not null"` // chat, image, video, embedding
	ProviderID   uuid.UUID `gorm:"type:uuid;not null"`
	ModelID      string    `gorm:"not null"`
	InputTokens  int       `gorm:"not null;default:0"`
	OutputTokens int       `gorm:"not null;default:0"`
	TotalTokens  int       `gorm:"not null;default:0"`
	CostUSD      float64   `gorm:"type:decimal(10,6);not null"`
	LatencyMs    int       `gorm:"not null"`
	Success      bool      `gorm:"not null"`
}

// TableName returns the database table name.
func (UsageRecord) TableName() string {
	return "usage_records"
}
