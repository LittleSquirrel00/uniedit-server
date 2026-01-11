package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uniedit/server/internal/utils/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID(t *testing.T) {
	t.Run("generates new request ID when not provided", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			requestID := GetRequestID(c)
			c.String(http.StatusOK, requestID)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Check response header
		headerID := w.Header().Get(RequestIDHeader)
		assert.NotEmpty(t, headerID)
		// Check body matches header
		assert.Equal(t, headerID, w.Body.String())
	})

	t.Run("uses existing request ID from header", func(t *testing.T) {
		router := gin.New()
		router.Use(RequestID())
		router.GET("/test", func(c *gin.Context) {
			requestID := GetRequestID(c)
			c.String(http.StatusOK, requestID)
		})

		existingID := "existing-request-id-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RequestIDHeader, existingID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, existingID, w.Header().Get(RequestIDHeader))
		assert.Equal(t, existingID, w.Body.String())
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns empty string when not set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		id := GetRequestID(c)
		assert.Empty(t, id)
	})

	t.Run("returns request ID when set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(RequestIDKey, "test-id")
		id := GetRequestID(c)
		assert.Equal(t, "test-id", id)
	})
}

func TestLogging(t *testing.T) {
	t.Run("logs successful requests", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(&logger.Config{
			Level:  "info",
			Format: "json",
			Output: buf,
		})

		router := gin.New()
		router.Use(RequestID())
		router.Use(Logging(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "HTTP Request")
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "/test")
		assert.Contains(t, logOutput, "200")
	})

	t.Run("logs 4xx requests as warnings", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(&logger.Config{
			Level:  "warn",
			Format: "json",
			Output: buf,
		})

		router := gin.New()
		router.Use(Logging(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusNotFound, "not found")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "404")
	})

	t.Run("logs 5xx requests as errors", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(&logger.Config{
			Level:  "error",
			Format: "json",
			Output: buf,
		})

		router := gin.New()
		router.Use(Logging(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "error")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		logOutput := buf.String()
		assert.Contains(t, logOutput, "ERROR")
		assert.Contains(t, logOutput, "500")
	})

	t.Run("includes query parameters", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(&logger.Config{
			Level:  "info",
			Format: "json",
			Output: buf,
		})

		router := gin.New()
		router.Use(Logging(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test?foo=bar", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Contains(t, buf.String(), "foo=bar")
	})

	t.Run("includes user agent", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(&logger.Config{
			Level:  "info",
			Format: "json",
			Output: buf,
		})

		router := gin.New()
		router.Use(Logging(log))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "TestAgent/1.0")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Contains(t, buf.String(), "TestAgent/1.0")
	})
}

func TestRecovery(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(&logger.Config{
			Level:  "error",
			Format: "json",
			Output: buf,
		})

		router := gin.New()
		router.Use(Recovery(log))
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		// Should not panic
		require.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")

		// Check logging
		logOutput := buf.String()
		assert.Contains(t, logOutput, "Panic recovered")
		assert.Contains(t, logOutput, "test panic")
	})

	t.Run("uses default logger when nil", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery(nil))
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		require.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCORS(t *testing.T) {
	t.Run("creates cors middleware without error", func(t *testing.T) {
		cfg := DefaultCORSConfig()
		middleware := CORS(cfg)
		assert.NotNil(t, middleware)
	})

	t.Run("custom config creates middleware", func(t *testing.T) {
		cfg := CORSConfig{
			AllowOrigins: []string{"http://allowed.com"},
			AllowMethods: []string{"GET"},
			AllowHeaders: []string{"Content-Type"},
		}
		middleware := CORS(cfg)
		assert.NotNil(t, middleware)
	})
}

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	assert.Equal(t, []string{"*"}, cfg.AllowOrigins)
	assert.Contains(t, cfg.AllowMethods, "GET")
	assert.Contains(t, cfg.AllowMethods, "POST")
	assert.Contains(t, cfg.AllowMethods, "PUT")
	assert.Contains(t, cfg.AllowMethods, "DELETE")
	assert.Contains(t, cfg.AllowHeaders, "Authorization")
	assert.Contains(t, cfg.AllowHeaders, "Content-Type")
	assert.False(t, cfg.AllowCredentials)
}
