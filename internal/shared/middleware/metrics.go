package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/shared/metrics"
)

// Metrics returns a middleware that records HTTP metrics.
func Metrics(m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath() // Use route pattern, not actual path
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// Track in-flight requests
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		status := c.Writer.Status()

		m.RecordHTTPRequest(method, path, status, duration)
	}
}
