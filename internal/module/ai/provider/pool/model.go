package pool

import (
	"time"

	"github.com/google/uuid"
)

// HealthStatus represents the health state of a provider account.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// IsHealthy returns true if the status is healthy.
func (s HealthStatus) IsHealthy() bool {
	return s == HealthStatusHealthy
}

// CanServeRequests returns true if the account can serve requests.
func (s HealthStatus) CanServeRequests() bool {
	return s == HealthStatusHealthy || s == HealthStatusDegraded
}

// ProviderAccount represents a single API key/account for a provider.
type ProviderAccount struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProviderID      uuid.UUID `json:"provider_id" gorm:"type:uuid;not null;index"`
	Name            string    `json:"name" gorm:"not null"`
	EncryptedAPIKey string    `json:"-" gorm:"column:encrypted_api_key;not null"`
	KeyPrefix       string    `json:"key_prefix" gorm:"not null"`

	// Scheduling
	Weight   int  `json:"weight" gorm:"default:1"`
	Priority int  `json:"priority" gorm:"default:0"`
	IsActive bool `json:"is_active" gorm:"default:true"`

	// Health monitoring
	HealthStatus        HealthStatus `json:"health_status" gorm:"default:'healthy'"`
	LastHealthCheck     *time.Time   `json:"last_health_check,omitempty"`
	ConsecutiveFailures int          `json:"consecutive_failures" gorm:"default:0"`
	LastFailureAt       *time.Time   `json:"last_failure_at,omitempty"`

	// Rate limits (0 = use provider default)
	RateLimitRPM int `json:"rate_limit_rpm" gorm:"default:0"`
	RateLimitTPM int `json:"rate_limit_tpm" gorm:"default:0"`
	DailyLimit   int `json:"daily_limit" gorm:"default:0"`

	// Usage statistics (denormalized)
	TotalRequests int64   `json:"total_requests" gorm:"default:0"`
	TotalTokens   int64   `json:"total_tokens" gorm:"default:0"`
	TotalCostUSD  float64 `json:"total_cost_usd" gorm:"type:decimal(12,6);default:0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Decrypted key (not stored, populated at runtime)
	DecryptedKey string `json:"-" gorm:"-"`
}

// TableName returns the database table name.
func (ProviderAccount) TableName() string {
	return "provider_accounts"
}

// IsAvailable returns true if the account can handle requests.
func (a *ProviderAccount) IsAvailable() bool {
	return a.IsActive && a.HealthStatus.CanServeRequests()
}

// AccountUsageStats represents daily usage statistics for an account.
type AccountUsageStats struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID     uuid.UUID `json:"account_id" gorm:"type:uuid;not null"`
	Date          time.Time `json:"date" gorm:"type:date;not null"`
	RequestsCount int64     `json:"requests_count" gorm:"default:0"`
	TokensCount   int64     `json:"tokens_count" gorm:"default:0"`
	CostUSD       float64   `json:"cost_usd" gorm:"type:decimal(12,6);default:0"`
	CreatedAt     time.Time `json:"created_at"`
}

// TableName returns the database table name.
func (AccountUsageStats) TableName() string {
	return "account_usage_stats"
}

// AccountStats represents aggregated statistics for an account.
type AccountStats struct {
	AccountID     uuid.UUID `json:"account_id"`
	TotalRequests int64     `json:"total_requests"`
	TotalTokens   int64     `json:"total_tokens"`
	TotalCostUSD  float64   `json:"total_cost_usd"`

	// Health info
	HealthStatus        HealthStatus `json:"health_status"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	LastHealthCheck     *time.Time   `json:"last_health_check,omitempty"`

	// Daily breakdown (last 30 days)
	DailyStats []AccountUsageStats `json:"daily_stats,omitempty"`
}

// Health transition thresholds.
const (
	// FailuresToDegrade is the number of consecutive failures to transition from healthy to degraded.
	FailuresToDegrade = 2
	// FailuresToUnhealthy is the number of consecutive failures to transition from degraded to unhealthy.
	FailuresToUnhealthy = 5
	// SuccessesToRecover is the number of consecutive successes to transition from degraded to healthy.
	SuccessesToRecover = 3
	// CircuitBreakerCooldown is the duration to wait before allowing a test request when unhealthy.
	CircuitBreakerCooldown = 30 * time.Second
	// LatencyThreshold is the latency threshold (in milliseconds) that triggers degradation.
	LatencyThreshold = 3000
)
