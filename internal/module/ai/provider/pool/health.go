package pool

import (
	"sync"
	"time"
)

// HealthMonitor manages health state transitions for accounts.
type HealthMonitor struct {
	mu sync.RWMutex
	// successCounts tracks consecutive successes for recovery
	successCounts map[string]int
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		successCounts: make(map[string]int),
	}
}

// RecordSuccess records a successful request and returns the new health status.
func (m *HealthMonitor) RecordSuccess(account *ProviderAccount) (HealthStatus, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := account.ID.String()
	currentStatus := account.HealthStatus
	failures := 0

	switch currentStatus {
	case HealthStatusHealthy:
		// Stay healthy
		m.successCounts[key] = 0
		return HealthStatusHealthy, 0

	case HealthStatusDegraded:
		// Count consecutive successes for recovery
		m.successCounts[key]++
		if m.successCounts[key] >= SuccessesToRecover {
			m.successCounts[key] = 0
			return HealthStatusHealthy, 0
		}
		return HealthStatusDegraded, 0

	case HealthStatusUnhealthy:
		// First success after cooldown → degraded
		m.successCounts[key] = 1
		return HealthStatusDegraded, 0

	default:
		return HealthStatusHealthy, failures
	}
}

// RecordFailure records a failed request and returns the new health status.
func (m *HealthMonitor) RecordFailure(account *ProviderAccount) (HealthStatus, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := account.ID.String()
	m.successCounts[key] = 0 // Reset success counter

	newFailures := account.ConsecutiveFailures + 1
	currentStatus := account.HealthStatus

	switch currentStatus {
	case HealthStatusHealthy:
		if newFailures >= FailuresToDegrade {
			return HealthStatusDegraded, newFailures
		}
		return HealthStatusHealthy, newFailures

	case HealthStatusDegraded:
		if newFailures >= FailuresToUnhealthy {
			return HealthStatusUnhealthy, newFailures
		}
		return HealthStatusDegraded, newFailures

	case HealthStatusUnhealthy:
		// Stay unhealthy
		return HealthStatusUnhealthy, newFailures

	default:
		return currentStatus, newFailures
	}
}

// RecordHighLatency records a high latency event and returns the new health status.
func (m *HealthMonitor) RecordHighLatency(account *ProviderAccount, latencyMs int) (HealthStatus, int) {
	if latencyMs < LatencyThreshold {
		return account.HealthStatus, account.ConsecutiveFailures
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := account.ID.String()
	m.successCounts[key] = 0

	currentStatus := account.HealthStatus

	// High latency degrades healthy accounts
	if currentStatus == HealthStatusHealthy {
		return HealthStatusDegraded, account.ConsecutiveFailures
	}

	return currentStatus, account.ConsecutiveFailures
}

// CanAttemptRequest checks if an unhealthy account can attempt a test request.
func (m *HealthMonitor) CanAttemptRequest(account *ProviderAccount) bool {
	if account.HealthStatus != HealthStatusUnhealthy {
		return true
	}

	// Check if cooldown period has passed
	if account.LastFailureAt == nil {
		return true
	}

	return time.Since(*account.LastFailureAt) >= CircuitBreakerCooldown
}

// Reset resets the health monitor state for an account.
func (m *HealthMonitor) Reset(accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.successCounts, accountID)
}
