package billing

import (
	"time"

	"github.com/google/uuid"
)

// GetPlansResponse represents the response for listing plans.
type GetPlansResponse struct {
	Plans []*PlanResponse `json:"plans"`
}

// PlanResponse represents a plan in API responses.
type PlanResponse struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	BillingCycle  string   `json:"billing_cycle,omitempty"`
	PriceUSD      int64    `json:"price_usd"`
	MonthlyTokens int64    `json:"monthly_tokens"`
	DailyRequests int      `json:"daily_requests"`
	MaxAPIKeys    int      `json:"max_api_keys"`
	Features      []string `json:"features"`
}

// ToResponse converts a Plan to PlanResponse.
func (p *Plan) ToResponse() *PlanResponse {
	features := make([]string, len(p.Features))
	copy(features, p.Features)
	return &PlanResponse{
		ID:            p.ID,
		Type:          string(p.Type),
		Name:          p.Name,
		Description:   p.Description,
		BillingCycle:  string(p.BillingCycle),
		PriceUSD:      p.PriceUSD,
		MonthlyTokens: p.MonthlyTokens,
		DailyRequests: p.DailyRequests,
		MaxAPIKeys:    p.MaxAPIKeys,
		Features:      features,
	}
}

// SubscriptionResponse represents a subscription in API responses.
type SubscriptionResponse struct {
	ID                 uuid.UUID `json:"id"`
	PlanID             string    `json:"plan_id"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
	CanceledAt         *time.Time `json:"canceled_at,omitempty"`
	CreditsBalance     int64     `json:"credits_balance"`
	CreatedAt          time.Time `json:"created_at"`
	Plan               *PlanResponse `json:"plan,omitempty"`
}

// ToResponse converts a Subscription to SubscriptionResponse.
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

// UsagePeriod represents the time period for usage queries.
type UsagePeriod string

const (
	UsagePeriodDay   UsagePeriod = "day"
	UsagePeriodWeek  UsagePeriod = "week"
	UsagePeriodMonth UsagePeriod = "month"
)

// UsageStatsRequest represents usage statistics query parameters.
type UsageStatsRequest struct {
	Period    UsagePeriod `form:"period" binding:"required,oneof=day week month"`
	StartDate *time.Time  `form:"start_date"`
	EndDate   *time.Time  `form:"end_date"`
}

// UsageStats represents aggregated usage statistics.
type UsageStats struct {
	TotalTokens   int64                `json:"total_tokens"`
	TotalRequests int                  `json:"total_requests"`
	TotalCostUSD  float64              `json:"total_cost_usd"`
	ByModel       map[string]*ModelUsage `json:"by_model"`
	ByDay         []*DailyUsage        `json:"by_day"`
}

// ModelUsage represents usage by model.
type ModelUsage struct {
	ModelID      string  `json:"model_id"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalRequests int    `json:"total_requests"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// DailyUsage represents usage for a single day.
type DailyUsage struct {
	Date          string  `json:"date"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalRequests int     `json:"total_requests"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
}

// CancelSubscriptionRequest represents a cancel subscription request.
type CancelSubscriptionRequest struct {
	Immediately bool `json:"immediately"`
}

// RecordUsageRequest represents a request to record usage.
type RecordUsageRequest struct {
	RequestID    string    `json:"request_id"`
	TaskType     string    `json:"task_type"`
	ProviderID   uuid.UUID `json:"provider_id"`
	ModelID      string    `json:"model_id"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	LatencyMs    int       `json:"latency_ms"`
	Success      bool      `json:"success"`
}
