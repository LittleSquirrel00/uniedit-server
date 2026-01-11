package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all application metrics.
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// AI metrics
	AIRequestsTotal    *prometheus.CounterVec
	AIRequestDuration  *prometheus.HistogramVec
	AITokensTotal      *prometheus.CounterVec
	AIProviderHealth   *prometheus.GaugeVec

	// Auth metrics
	AuthEventsTotal *prometheus.CounterVec
	ActiveSessions  prometheus.Gauge

	// Database metrics
	DBQueryDuration *prometheus.HistogramVec
	DBConnectionsOpen prometheus.Gauge

	// Cache metrics
	CacheHitsTotal   *prometheus.CounterVec
	CacheMissesTotal *prometheus.CounterVec
}

// New creates a new Metrics instance with all metrics registered.
func New(namespace string) *Metrics {
	if namespace == "" {
		namespace = "uniedit"
	}

	return &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
		),

		// AI metrics
		AIRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "requests_total",
				Help:      "Total number of AI requests",
			},
			[]string{"provider", "model", "status"},
		),
		AIRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "request_duration_seconds",
				Help:      "AI request duration in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120},
			},
			[]string{"provider", "model"},
		),
		AITokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "tokens_total",
				Help:      "Total number of tokens processed",
			},
			[]string{"provider", "model", "type"}, // type: input, output
		),
		AIProviderHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "ai",
				Name:      "provider_health",
				Help:      "Provider health status (1=healthy, 0=unhealthy)",
			},
			[]string{"provider"},
		),

		// Auth metrics
		AuthEventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "events_total",
				Help:      "Total number of auth events",
			},
			[]string{"event", "provider"}, // event: login_success, login_failed, logout, token_refresh
		),
		ActiveSessions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "auth",
				Name:      "active_sessions",
				Help:      "Number of active user sessions",
			},
		),

		// Database metrics
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation"}, // select, insert, update, delete
		),
		DBConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "connections_open",
				Help:      "Number of open database connections",
			},
		),

		// Cache metrics
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "cache",
				Name:      "hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache"},
		),
		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "cache",
				Name:      "misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache"},
		),
	}
}

// --- Convenience methods ---

// RecordHTTPRequest records an HTTP request.
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	statusStr := statusCodeToString(status)
	m.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordAIRequest records an AI request.
func (m *Metrics) RecordAIRequest(provider, model, status string, duration time.Duration) {
	m.AIRequestsTotal.WithLabelValues(provider, model, status).Inc()
	m.AIRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordAITokens records token usage.
func (m *Metrics) RecordAITokens(provider, model string, inputTokens, outputTokens int) {
	if inputTokens > 0 {
		m.AITokensTotal.WithLabelValues(provider, model, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		m.AITokensTotal.WithLabelValues(provider, model, "output").Add(float64(outputTokens))
	}
}

// SetProviderHealth sets the health status of a provider.
func (m *Metrics) SetProviderHealth(provider string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.AIProviderHealth.WithLabelValues(provider).Set(value)
}

// RecordAuthEvent records an auth event.
func (m *Metrics) RecordAuthEvent(event, provider string) {
	m.AuthEventsTotal.WithLabelValues(event, provider).Inc()
}

// RecordDBQuery records a database query.
func (m *Metrics) RecordDBQuery(operation string, duration time.Duration) {
	m.DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit.
func (m *Metrics) RecordCacheHit(cache string) {
	m.CacheHitsTotal.WithLabelValues(cache).Inc()
}

// RecordCacheMiss records a cache miss.
func (m *Metrics) RecordCacheMiss(cache string) {
	m.CacheMissesTotal.WithLabelValues(cache).Inc()
}

// statusCodeToString converts an HTTP status code to a string category.
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
