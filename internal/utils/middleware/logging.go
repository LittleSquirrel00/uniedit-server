package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/utils/logger"
)

// Logging returns a middleware that logs HTTP requests.
func Logging(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		// Build log entry
		attrs := []any{
			"status", status,
			"method", method,
			"path", path,
			"latency_ms", latency.Milliseconds(),
			"client_ip", clientIP,
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if userAgent != "" {
			attrs = append(attrs, "user_agent", userAgent)
		}

		// Add request ID if present
		if requestID := c.GetString("request_id"); requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}

		// Add user ID if present
		if userID, exists := c.Get("user_id"); exists {
			attrs = append(attrs, "user_id", userID)
		}

		// Add error if present
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		// Log based on status code
		msg := "HTTP Request"
		switch {
		case status >= 500:
			log.Error(msg, attrs...)
		case status >= 400:
			log.Warn(msg, attrs...)
		default:
			log.Info(msg, attrs...)
		}
	}
}
