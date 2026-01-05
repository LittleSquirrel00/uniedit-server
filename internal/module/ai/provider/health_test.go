package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
)

// MockHealthChecker implements HealthChecker for testing.
type MockHealthChecker struct {
	healthyProviders map[uuid.UUID]bool
	err              error
}

func (m *MockHealthChecker) HealthCheck(_ context.Context, prov *Provider) error {
	if m.err != nil {
		return m.err
	}
	if healthy, ok := m.healthyProviders[prov.ID]; ok && !healthy {
		return errors.New("provider unhealthy")
	}
	return nil
}

func newTestRegistry() *Registry {
	p1ID := uuid.New()
	providers := []*Provider{
		{
			ID:      p1ID,
			Name:    "OpenAI",
			Type:    ProviderTypeOpenAI,
			Enabled: true,
			Models: []*Model{
				{
					ID:            "gpt-4o",
					ProviderID:    p1ID,
					Name:          "GPT-4o",
					Capabilities:  pq.StringArray{"chat"},
					ContextWindow: 128000,
					Enabled:       true,
				},
			},
		},
	}

	repo := &MockRepository{providers: providers}
	registry := NewRegistry(repo, nil)
	_ = registry.Refresh(context.Background())
	return registry
}

func TestDefaultHealthMonitorConfig(t *testing.T) {
	config := DefaultHealthMonitorConfig()
	assert.NotNil(t, config)
	assert.NotZero(t, config.CheckInterval)
	assert.NotZero(t, config.FailureThreshold)
	assert.NotZero(t, config.SuccessThreshold)
	assert.NotZero(t, config.Timeout)
	assert.NotZero(t, config.MaxHalfOpenRequests)
}

func TestNewHealthMonitor(t *testing.T) {
	t.Run("Creates with default config", func(t *testing.T) {
		registry := newTestRegistry()
		checker := &MockHealthChecker{}
		monitor := NewHealthMonitor(registry, checker, nil)

		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.breakers)
		assert.NotNil(t, monitor.healthStatus)
	})

	t.Run("Creates with custom config", func(t *testing.T) {
		registry := newTestRegistry()
		checker := &MockHealthChecker{}
		config := &HealthMonitorConfig{CheckInterval: 10}
		monitor := NewHealthMonitor(registry, checker, config)

		assert.NotNil(t, monitor)
		assert.Equal(t, config.CheckInterval, monitor.checkInterval)
	})
}

func TestHealthMonitor_Start(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)

	err := monitor.Start(context.Background())
	assert.NoError(t, err)

	// All providers should be healthy initially
	for _, p := range registry.AllProviders() {
		assert.True(t, monitor.IsHealthy(p.ID))
	}

	monitor.Stop()
}

func TestHealthMonitor_CheckProvider(t *testing.T) {
	registry := newTestRegistry()
	providers := registry.AllProviders()

	t.Run("Healthy provider", func(t *testing.T) {
		checker := &MockHealthChecker{}
		monitor := NewHealthMonitor(registry, checker, nil)

		err := monitor.CheckProvider(context.Background(), providers[0])
		assert.NoError(t, err)
		assert.Equal(t, HealthStatusHealthy, monitor.GetStatus(providers[0].ID))
	})

	t.Run("Unhealthy provider", func(t *testing.T) {
		checker := &MockHealthChecker{err: errors.New("connection failed")}
		monitor := NewHealthMonitor(registry, checker, nil)

		err := monitor.CheckProvider(context.Background(), providers[0])
		assert.Error(t, err)
		assert.Equal(t, HealthStatusUnhealthy, monitor.GetStatus(providers[0].ID))
	})
}

func TestHealthMonitor_IsHealthy(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)

	providers := registry.AllProviders()

	t.Run("Unknown provider is healthy", func(t *testing.T) {
		assert.True(t, monitor.IsHealthy(uuid.New()))
	})

	t.Run("Checked healthy provider", func(t *testing.T) {
		_ = monitor.CheckProvider(context.Background(), providers[0])
		assert.True(t, monitor.IsHealthy(providers[0].ID))
	})

	t.Run("Checked unhealthy provider", func(t *testing.T) {
		checker.err = errors.New("failed")
		_ = monitor.CheckProvider(context.Background(), providers[0])
		assert.False(t, monitor.IsHealthy(providers[0].ID))
	})
}

func TestHealthMonitor_GetStatus(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)

	t.Run("Unknown provider returns healthy", func(t *testing.T) {
		status := monitor.GetStatus(uuid.New())
		assert.Equal(t, HealthStatusHealthy, status)
	})

	t.Run("Returns correct status after check", func(t *testing.T) {
		providers := registry.AllProviders()
		_ = monitor.CheckProvider(context.Background(), providers[0])
		assert.Equal(t, HealthStatusHealthy, monitor.GetStatus(providers[0].ID))
	})
}

func TestHealthMonitor_GetBreakerState(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)

	t.Run("New breaker is closed", func(t *testing.T) {
		state := monitor.GetBreakerState(uuid.New())
		assert.Equal(t, gobreaker.StateClosed, state)
	})
}

func TestHealthMonitor_IsBreakerOpen(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)

	t.Run("New breaker is not open", func(t *testing.T) {
		assert.False(t, monitor.IsBreakerOpen(uuid.New()))
	})
}

func TestHealthMonitor_Execute(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)
	providerID := uuid.New()

	t.Run("Successful execution", func(t *testing.T) {
		executed := false
		err := monitor.Execute(providerID, func() error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("Failed execution", func(t *testing.T) {
		err := monitor.Execute(providerID, func() error {
			return errors.New("execution failed")
		})

		assert.Error(t, err)
	})
}

func TestHealthMonitor_AllHealthStatus(t *testing.T) {
	registry := newTestRegistry()
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(registry, checker, nil)

	// Start to initialize health status
	_ = monitor.Start(context.Background())
	defer monitor.Stop()

	status := monitor.AllHealthStatus()
	assert.NotNil(t, status)
	assert.Equal(t, len(registry.AllProviders()), len(status))
}

func TestHealthStatus(t *testing.T) {
	assert.Equal(t, HealthStatus("healthy"), HealthStatusHealthy)
	assert.Equal(t, HealthStatus("degraded"), HealthStatusDegraded)
	assert.Equal(t, HealthStatus("unhealthy"), HealthStatusUnhealthy)
}
