package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// createTestMetrics creates metrics with a custom registry for testing.
// This avoids conflicts with the default registry.
func createTestMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "test"
	}

	reg := prometheus.NewRegistry()

	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
		),
		AIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "requests_total",
				Help:      "Total number of AI requests",
			},
			[]string{"provider", "model", "status"},
		),
		AIRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "request_duration_seconds",
				Help:      "AI request duration in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120},
			},
			[]string{"provider", "model"},
		),
		AITokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "tokens_total",
				Help:      "Total number of tokens processed",
			},
			[]string{"provider", "model", "type"},
		),
		AIProviderHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "provider_health",
				Help:      "Provider health status (1=healthy, 0=unhealthy)",
			},
			[]string{"provider"},
		),
		AuthEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "events_total",
				Help:      "Total number of auth events",
			},
			[]string{"event", "provider"},
		),
		ActiveSessions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "active_sessions",
				Help:      "Number of active user sessions",
			},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation"},
		),
		DBConnectionsOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "connections_open",
				Help:      "Number of open database connections",
			},
		),
		CacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "cache",
				Name:      "hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache"},
		),
		CacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "cache",
				Name:      "misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache"},
		),
	}

	// Register with test registry
	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.AIRequestsTotal,
		m.AIRequestDuration,
		m.AITokensTotal,
		m.AIProviderHealth,
		m.AuthEventsTotal,
		m.ActiveSessions,
		m.DBQueryDuration,
		m.DBConnectionsOpen,
		m.CacheHitsTotal,
		m.CacheMissesTotal,
	)

	return m
}

func TestNew(t *testing.T) {
	t.Run("creates with default namespace", func(t *testing.T) {
		// Note: This test may fail if run multiple times in the same process
		// due to prometheus global registry. In practice, use createTestMetrics.
		m := New("test_new")
		assert.NotNil(t, m)
		assert.NotNil(t, m.HTTPRequestsTotal)
		assert.NotNil(t, m.HTTPRequestDuration)
		assert.NotNil(t, m.HTTPRequestsInFlight)
		assert.NotNil(t, m.AIRequestsTotal)
		assert.NotNil(t, m.AIRequestDuration)
		assert.NotNil(t, m.AITokensTotal)
		assert.NotNil(t, m.AIProviderHealth)
		assert.NotNil(t, m.AuthEventsTotal)
		assert.NotNil(t, m.ActiveSessions)
		assert.NotNil(t, m.DBQueryDuration)
		assert.NotNil(t, m.DBConnectionsOpen)
		assert.NotNil(t, m.CacheHitsTotal)
		assert.NotNil(t, m.CacheMissesTotal)
	})
}

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	m := createTestMetrics("http_test")

	t.Run("records request with 2xx status", func(t *testing.T) {
		m.RecordHTTPRequest("GET", "/api/users", 200, 100*time.Millisecond)

		count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/api/users", "2xx"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records request with 4xx status", func(t *testing.T) {
		m.RecordHTTPRequest("POST", "/api/auth", 401, 50*time.Millisecond)

		count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("POST", "/api/auth", "4xx"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records request with 5xx status", func(t *testing.T) {
		m.RecordHTTPRequest("PUT", "/api/data", 500, 200*time.Millisecond)

		count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("PUT", "/api/data", "5xx"))
		assert.Equal(t, float64(1), count)
	})
}

func TestMetrics_RecordAIRequest(t *testing.T) {
	m := createTestMetrics("ai_test")

	t.Run("records successful AI request", func(t *testing.T) {
		m.RecordAIRequest("openai", "gpt-4o", "success", 2*time.Second)

		count := testutil.ToFloat64(m.AIRequestsTotal.WithLabelValues("openai", "gpt-4o", "success"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records failed AI request", func(t *testing.T) {
		m.RecordAIRequest("anthropic", "claude-3", "error", 500*time.Millisecond)

		count := testutil.ToFloat64(m.AIRequestsTotal.WithLabelValues("anthropic", "claude-3", "error"))
		assert.Equal(t, float64(1), count)
	})
}

func TestMetrics_RecordAITokens(t *testing.T) {
	m := createTestMetrics("tokens_test")

	t.Run("records input and output tokens", func(t *testing.T) {
		m.RecordAITokens("openai", "gpt-4o", 100, 50)

		inputCount := testutil.ToFloat64(m.AITokensTotal.WithLabelValues("openai", "gpt-4o", "input"))
		outputCount := testutil.ToFloat64(m.AITokensTotal.WithLabelValues("openai", "gpt-4o", "output"))

		assert.Equal(t, float64(100), inputCount)
		assert.Equal(t, float64(50), outputCount)
	})

	t.Run("skips zero tokens", func(t *testing.T) {
		m.RecordAITokens("openai", "gpt-4o-mini", 0, 0)

		// These should still be 0 (not registered at all for these labels)
		inputCount := testutil.ToFloat64(m.AITokensTotal.WithLabelValues("openai", "gpt-4o-mini", "input"))
		outputCount := testutil.ToFloat64(m.AITokensTotal.WithLabelValues("openai", "gpt-4o-mini", "output"))

		assert.Equal(t, float64(0), inputCount)
		assert.Equal(t, float64(0), outputCount)
	})
}

func TestMetrics_SetProviderHealth(t *testing.T) {
	m := createTestMetrics("health_test")

	t.Run("sets provider as healthy", func(t *testing.T) {
		m.SetProviderHealth("openai", true)

		health := testutil.ToFloat64(m.AIProviderHealth.WithLabelValues("openai"))
		assert.Equal(t, float64(1), health)
	})

	t.Run("sets provider as unhealthy", func(t *testing.T) {
		m.SetProviderHealth("anthropic", false)

		health := testutil.ToFloat64(m.AIProviderHealth.WithLabelValues("anthropic"))
		assert.Equal(t, float64(0), health)
	})
}

func TestMetrics_RecordAuthEvent(t *testing.T) {
	m := createTestMetrics("auth_test")

	t.Run("records login success", func(t *testing.T) {
		m.RecordAuthEvent("login_success", "github")

		count := testutil.ToFloat64(m.AuthEventsTotal.WithLabelValues("login_success", "github"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records login failure", func(t *testing.T) {
		m.RecordAuthEvent("login_failed", "google")

		count := testutil.ToFloat64(m.AuthEventsTotal.WithLabelValues("login_failed", "google"))
		assert.Equal(t, float64(1), count)
	})
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	m := createTestMetrics("db_test")

	t.Run("records select query", func(t *testing.T) {
		m.RecordDBQuery("select", 10*time.Millisecond)

		// Histogram observations are harder to test, just verify no panic
	})

	t.Run("records insert query", func(t *testing.T) {
		m.RecordDBQuery("insert", 5*time.Millisecond)
	})
}

func TestMetrics_RecordCache(t *testing.T) {
	m := createTestMetrics("cache_test")

	t.Run("records cache hit", func(t *testing.T) {
		m.RecordCacheHit("users")

		count := testutil.ToFloat64(m.CacheHitsTotal.WithLabelValues("users"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records cache miss", func(t *testing.T) {
		m.RecordCacheMiss("sessions")

		count := testutil.ToFloat64(m.CacheMissesTotal.WithLabelValues("sessions"))
		assert.Equal(t, float64(1), count)
	})
}

func TestStatusCodeToString(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{301, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{599, "5xx"},
		{100, "unknown"},
		{0, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.code)), func(t *testing.T) {
			result := statusCodeToString(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}
