package provider

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sony/gobreaker/v2"
)

// HealthStatus represents the health status of a provider.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthChecker defines the interface for health checking.
type HealthChecker interface {
	HealthCheck(ctx context.Context, prov *Provider) error
}

// HealthMonitor monitors provider health.
type HealthMonitor struct {
	mu sync.RWMutex

	registry       *Registry
	checker        HealthChecker
	breakers       map[uuid.UUID]*gobreaker.CircuitBreaker[any]
	healthStatus   map[uuid.UUID]HealthStatus
	lastCheck      map[uuid.UUID]time.Time

	checkInterval  time.Duration
	stopMonitor    chan struct{}
}

// HealthMonitorConfig contains health monitor configuration.
type HealthMonitorConfig struct {
	CheckInterval       time.Duration
	FailureThreshold    uint32
	SuccessThreshold    uint32
	Timeout             time.Duration
	MaxHalfOpenRequests uint32
}

// DefaultHealthMonitorConfig returns the default health monitor configuration.
func DefaultHealthMonitorConfig() *HealthMonitorConfig {
	return &HealthMonitorConfig{
		CheckInterval:       30 * time.Second,
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             60 * time.Second,
		MaxHalfOpenRequests: 1,
	}
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(registry *Registry, checker HealthChecker, config *HealthMonitorConfig) *HealthMonitor {
	if config == nil {
		config = DefaultHealthMonitorConfig()
	}

	return &HealthMonitor{
		registry:      registry,
		checker:       checker,
		breakers:      make(map[uuid.UUID]*gobreaker.CircuitBreaker[any]),
		healthStatus:  make(map[uuid.UUID]HealthStatus),
		lastCheck:     make(map[uuid.UUID]time.Time),
		checkInterval: config.CheckInterval,
		stopMonitor:   make(chan struct{}),
	}
}

// Start starts the health monitor.
func (m *HealthMonitor) Start(ctx context.Context) error {
	// Initialize circuit breakers for all providers
	for _, p := range m.registry.AllProviders() {
		m.getOrCreateBreaker(p.ID)
		m.healthStatus[p.ID] = HealthStatusHealthy
	}

	// Start background health check
	go m.monitorLoop()

	return nil
}

// Stop stops the health monitor.
func (m *HealthMonitor) Stop() {
	close(m.stopMonitor)
}

// monitorLoop periodically checks provider health.
func (m *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopMonitor:
			return
		case <-ticker.C:
			m.checkAllProviders()
		}
	}
}

// checkAllProviders checks health of all providers.
func (m *HealthMonitor) checkAllProviders() {
	providers := m.registry.AllProviders()

	for _, p := range providers {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		m.CheckProvider(ctx, p)
		cancel()
	}
}

// CheckProvider checks the health of a specific provider.
func (m *HealthMonitor) CheckProvider(ctx context.Context, p *Provider) error {
	breaker := m.getOrCreateBreaker(p.ID)

	_, err := breaker.Execute(func() (any, error) {
		return nil, m.checker.HealthCheck(ctx, p)
	})

	m.mu.Lock()
	m.lastCheck[p.ID] = time.Now()
	if err != nil {
		m.healthStatus[p.ID] = HealthStatusUnhealthy
	} else {
		m.healthStatus[p.ID] = HealthStatusHealthy
	}
	m.mu.Unlock()

	return err
}

// IsHealthy checks if a provider is healthy.
func (m *HealthMonitor) IsHealthy(providerID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.healthStatus[providerID]
	if !ok {
		return true // Unknown providers are considered healthy
	}
	return status == HealthStatusHealthy
}

// GetStatus returns the health status of a provider.
func (m *HealthMonitor) GetStatus(providerID uuid.UUID) HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.healthStatus[providerID]
	if !ok {
		return HealthStatusHealthy
	}
	return status
}

// GetBreakerState returns the circuit breaker state for a provider.
func (m *HealthMonitor) GetBreakerState(providerID uuid.UUID) gobreaker.State {
	breaker := m.getOrCreateBreaker(providerID)
	return breaker.State()
}

// IsBreakerOpen checks if the circuit breaker is open.
func (m *HealthMonitor) IsBreakerOpen(providerID uuid.UUID) bool {
	return m.GetBreakerState(providerID) == gobreaker.StateOpen
}

// Execute executes a function with circuit breaker protection.
func (m *HealthMonitor) Execute(providerID uuid.UUID, fn func() error) error {
	breaker := m.getOrCreateBreaker(providerID)

	_, err := breaker.Execute(func() (any, error) {
		return nil, fn()
	})

	return err
}

// getOrCreateBreaker gets or creates a circuit breaker for a provider.
func (m *HealthMonitor) getOrCreateBreaker(providerID uuid.UUID) *gobreaker.CircuitBreaker[any] {
	m.mu.Lock()
	defer m.mu.Unlock()

	if breaker, ok := m.breakers[providerID]; ok {
		return breaker
	}

	settings := gobreaker.Settings{
		Name:        providerID.String(),
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}

	breaker := gobreaker.NewCircuitBreaker[any](settings)
	m.breakers[providerID] = breaker

	return breaker
}

// AllHealthStatus returns the health status of all providers.
func (m *HealthMonitor) AllHealthStatus() map[uuid.UUID]HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uuid.UUID]HealthStatus, len(m.healthStatus))
	for k, v := range m.healthStatus {
		result[k] = v
	}
	return result
}
