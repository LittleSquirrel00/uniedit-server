package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/utils/logger"
)

// Recovery returns a middleware that recovers from panics.
// If log is nil, it will use a default logger.
func Recovery(log *logger.Logger) gin.HandlerFunc {
	if log == nil {
		log = logger.New(nil)
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic
				log.Error("Panic recovered",
					"error", err,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
					"stack", stack,
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()
		c.Next()
	}
}
