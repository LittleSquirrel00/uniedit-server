package pool

import (
	"time"

	"github.com/google/uuid"
)

// CreateAccountRequest represents a request to add an account to the pool.
type CreateAccountRequest struct {
	Name         string `json:"name" binding:"required"`
	APIKey       string `json:"api_key" binding:"required"`
	Weight       int    `json:"weight,omitempty"`
	Priority     int    `json:"priority,omitempty"`
	RateLimitRPM int    `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM int    `json:"rate_limit_tpm,omitempty"`
	DailyLimit   int    `json:"daily_limit,omitempty"`
}

// UpdateAccountRequest represents a request to update an account.
type UpdateAccountRequest struct {
	Name         *string `json:"name,omitempty"`
	Weight       *int    `json:"weight,omitempty"`
	Priority     *int    `json:"priority,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
	RateLimitRPM *int    `json:"rate_limit_rpm,omitempty"`
	RateLimitTPM *int    `json:"rate_limit_tpm,omitempty"`
	DailyLimit   *int    `json:"daily_limit,omitempty"`
}

// AccountResponse represents an account in API responses.
type AccountResponse struct {
	ID                  uuid.UUID    `json:"id"`
	ProviderID          uuid.UUID    `json:"provider_id"`
	Name                string       `json:"name"`
	KeyPrefix           string       `json:"key_prefix"`
	Weight              int          `json:"weight"`
	Priority            int          `json:"priority"`
	IsActive            bool         `json:"is_active"`
	HealthStatus        HealthStatus `json:"health_status"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	LastHealthCheck     *time.Time   `json:"last_health_check,omitempty"`
	LastFailureAt       *time.Time   `json:"last_failure_at,omitempty"`
	RateLimitRPM        int          `json:"rate_limit_rpm"`
	RateLimitTPM        int          `json:"rate_limit_tpm"`
	DailyLimit          int          `json:"daily_limit"`
	TotalRequests       int64        `json:"total_requests"`
	TotalTokens         int64        `json:"total_tokens"`
	TotalCostUSD        float64      `json:"total_cost_usd"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// ToResponse converts ProviderAccount to AccountResponse.
func (a *ProviderAccount) ToResponse() *AccountResponse {
	return &AccountResponse{
		ID:                  a.ID,
		ProviderID:          a.ProviderID,
		Name:                a.Name,
		KeyPrefix:           a.KeyPrefix,
		Weight:              a.Weight,
		Priority:            a.Priority,
		IsActive:            a.IsActive,
		HealthStatus:        a.HealthStatus,
		ConsecutiveFailures: a.ConsecutiveFailures,
		LastHealthCheck:     a.LastHealthCheck,
		LastFailureAt:       a.LastFailureAt,
		RateLimitRPM:        a.RateLimitRPM,
		RateLimitTPM:        a.RateLimitTPM,
		DailyLimit:          a.DailyLimit,
		TotalRequests:       a.TotalRequests,
		TotalTokens:         a.TotalTokens,
		TotalCostUSD:        a.TotalCostUSD,
		CreatedAt:           a.CreatedAt,
		UpdatedAt:           a.UpdatedAt,
	}
}

// AccountStatsResponse represents account statistics in API responses.
type AccountStatsResponse struct {
	AccountID           uuid.UUID              `json:"account_id"`
	TotalRequests       int64                  `json:"total_requests"`
	TotalTokens         int64                  `json:"total_tokens"`
	TotalCostUSD        float64                `json:"total_cost_usd"`
	HealthStatus        HealthStatus           `json:"health_status"`
	ConsecutiveFailures int                    `json:"consecutive_failures"`
	LastHealthCheck     *time.Time             `json:"last_health_check,omitempty"`
	DailyStats          []DailyStatsResponse   `json:"daily_stats"`
}

// DailyStatsResponse represents daily usage statistics.
type DailyStatsResponse struct {
	Date          string  `json:"date"`
	RequestsCount int64   `json:"requests_count"`
	TokensCount   int64   `json:"tokens_count"`
	CostUSD       float64 `json:"cost_usd"`
}

// ToStatsResponse converts AccountStats to AccountStatsResponse.
func (s *AccountStats) ToResponse() *AccountStatsResponse {
	resp := &AccountStatsResponse{
		AccountID:           s.AccountID,
		TotalRequests:       s.TotalRequests,
		TotalTokens:         s.TotalTokens,
		TotalCostUSD:        s.TotalCostUSD,
		HealthStatus:        s.HealthStatus,
		ConsecutiveFailures: s.ConsecutiveFailures,
		LastHealthCheck:     s.LastHealthCheck,
		DailyStats:          make([]DailyStatsResponse, 0, len(s.DailyStats)),
	}

	for _, ds := range s.DailyStats {
		resp.DailyStats = append(resp.DailyStats, DailyStatsResponse{
			Date:          ds.Date.Format("2006-01-02"),
			RequestsCount: ds.RequestsCount,
			TokensCount:   ds.TokensCount,
			CostUSD:       ds.CostUSD,
		})
	}

	return resp
}

// ListAccountsResponse represents a list of accounts.
type ListAccountsResponse struct {
	Accounts []*AccountResponse `json:"accounts"`
	Total    int                `json:"total"`
}
