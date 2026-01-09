package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError(t *testing.T) {
	t.Run("Error returns message", func(t *testing.T) {
		err := &AppError{
			Code:    "TEST_ERROR",
			Message: "test error message",
		}
		assert.Equal(t, "test error message", err.Error())
	})

	t.Run("Error includes wrapped error", func(t *testing.T) {
		wrapped := errors.New("wrapped error")
		err := &AppError{
			Code:    "TEST_ERROR",
			Message: "test error message",
			Err:     wrapped,
		}
		assert.Contains(t, err.Error(), "test error message")
		assert.Contains(t, err.Error(), "wrapped error")
	})

	t.Run("Unwrap returns wrapped error", func(t *testing.T) {
		wrapped := errors.New("wrapped error")
		err := &AppError{
			Code:    "TEST_ERROR",
			Message: "test message",
			Err:     wrapped,
		}
		assert.Equal(t, wrapped, err.Unwrap())
	})
}

func TestNewAppError(t *testing.T) {
	wrapped := errors.New("original")
	err := NewAppError("CUSTOM_ERROR", "custom message", 418, wrapped)

	assert.Equal(t, "CUSTOM_ERROR", err.Code)
	assert.Equal(t, "custom message", err.Message)
	assert.Equal(t, 418, err.StatusCode)
	assert.Equal(t, wrapped, err.Err)
}

func TestNotFound(t *testing.T) {
	err := NotFound("user")

	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Equal(t, "user not found", err.Message)
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestUnauthorized(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		err := Unauthorized("invalid token")
		assert.Equal(t, "UNAUTHORIZED", err.Code)
		assert.Equal(t, "invalid token", err.Message)
		assert.Equal(t, http.StatusUnauthorized, err.StatusCode)
	})

	t.Run("with empty message uses default", func(t *testing.T) {
		err := Unauthorized("")
		assert.Equal(t, "authentication required", err.Message)
	})
}

func TestForbidden(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		err := Forbidden("insufficient permissions")
		assert.Equal(t, "FORBIDDEN", err.Code)
		assert.Equal(t, "insufficient permissions", err.Message)
		assert.Equal(t, http.StatusForbidden, err.StatusCode)
	})

	t.Run("with empty message uses default", func(t *testing.T) {
		err := Forbidden("")
		assert.Equal(t, "access denied", err.Message)
	})
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("invalid input")

	assert.Equal(t, "BAD_REQUEST", err.Code)
	assert.Equal(t, "invalid input", err.Message)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
}

func TestValidationError(t *testing.T) {
	err := ValidationError("field 'email' is required")

	assert.Equal(t, "VALIDATION_ERROR", err.Code)
	assert.Equal(t, "field 'email' is required", err.Message)
	assert.Equal(t, http.StatusUnprocessableEntity, err.StatusCode)
}

func TestConflict(t *testing.T) {
	err := Conflict("resource already exists")

	assert.Equal(t, "CONFLICT", err.Code)
	assert.Equal(t, "resource already exists", err.Message)
	assert.Equal(t, http.StatusConflict, err.StatusCode)
}

func TestInternal(t *testing.T) {
	wrapped := errors.New("database error")
	err := Internal("operation failed", wrapped)

	assert.Equal(t, "INTERNAL_ERROR", err.Code)
	assert.Equal(t, "operation failed", err.Message)
	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.Equal(t, wrapped, err.Err)
}

func TestQuotaExceeded(t *testing.T) {
	err := QuotaExceeded("API call limit reached")

	assert.Equal(t, "QUOTA_EXCEEDED", err.Code)
	assert.Equal(t, "API call limit reached", err.Message)
	assert.Equal(t, http.StatusPaymentRequired, err.StatusCode)
}

func TestRateLimited(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		err := RateLimited("slow down")
		assert.Equal(t, "RATE_LIMITED", err.Code)
		assert.Equal(t, "slow down", err.Message)
		assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	})

	t.Run("with empty message uses default", func(t *testing.T) {
		err := RateLimited("")
		assert.Equal(t, "too many requests", err.Message)
	})
}

func TestTimeout(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		err := Timeout("upstream timeout")
		assert.Equal(t, "TIMEOUT", err.Code)
		assert.Equal(t, "upstream timeout", err.Message)
		assert.Equal(t, http.StatusGatewayTimeout, err.StatusCode)
	})

	t.Run("with empty message uses default", func(t *testing.T) {
		err := Timeout("")
		assert.Equal(t, "request timeout", err.Message)
	})
}

func TestServiceUnavailable(t *testing.T) {
	t.Run("with custom message", func(t *testing.T) {
		err := ServiceUnavailable("under maintenance")
		assert.Equal(t, "SERVICE_UNAVAILABLE", err.Code)
		assert.Equal(t, "under maintenance", err.Message)
		assert.Equal(t, http.StatusServiceUnavailable, err.StatusCode)
	})

	t.Run("with empty message uses default", func(t *testing.T) {
		err := ServiceUnavailable("")
		assert.Equal(t, "service temporarily unavailable", err.Message)
	})
}

func TestToResponse(t *testing.T) {
	err := &AppError{
		Code:    "TEST_ERROR",
		Message: "test message",
	}

	resp := err.ToResponse()

	assert.Equal(t, "TEST_ERROR", resp.Error.Code)
	assert.Equal(t, "test message", resp.Error.Message)
}

func TestGetStatusCode(t *testing.T) {
	t.Run("from AppError", func(t *testing.T) {
		err := NotFound("resource")
		assert.Equal(t, http.StatusNotFound, GetStatusCode(err))
	})

	t.Run("from sentinel errors", func(t *testing.T) {
		tests := []struct {
			err      error
			expected int
		}{
			{ErrNotFound, http.StatusNotFound},
			{ErrUnauthorized, http.StatusUnauthorized},
			{ErrForbidden, http.StatusForbidden},
			{ErrBadRequest, http.StatusBadRequest},
			{ErrConflict, http.StatusConflict},
			{ErrQuotaExceeded, http.StatusPaymentRequired},
			{ErrRateLimited, http.StatusTooManyRequests},
			{ErrTimeout, http.StatusGatewayTimeout},
			{ErrServiceUnavail, http.StatusServiceUnavailable},
		}

		for _, tt := range tests {
			t.Run(tt.err.Error(), func(t *testing.T) {
				assert.Equal(t, tt.expected, GetStatusCode(tt.err))
			})
		}
	})

	t.Run("unknown error returns 500", func(t *testing.T) {
		err := errors.New("unknown error")
		assert.Equal(t, http.StatusInternalServerError, GetStatusCode(err))
	})
}

func TestWithDetails(t *testing.T) {
	err := BadRequest("validation failed")
	details := map[string]any{
		"field": "email",
		"value": "invalid",
	}

	result := err.WithDetails(details)

	assert.Same(t, err, result) // Returns same instance
	assert.Equal(t, details, err.Details)
}

func TestWithError(t *testing.T) {
	err := Internal("operation failed", nil)
	wrapped := errors.New("database connection lost")

	result := err.WithError(wrapped)

	assert.Same(t, err, result)
	assert.Equal(t, wrapped, err.Err)
}

func TestAppError_Is(t *testing.T) {
	t.Run("matches same code", func(t *testing.T) {
		err1 := &AppError{Code: "NOT_FOUND", Message: "user not found"}
		err2 := &AppError{Code: "NOT_FOUND", Message: "item not found"}

		assert.True(t, err1.Is(err2))
	})

	t.Run("does not match different code", func(t *testing.T) {
		err1 := &AppError{Code: "NOT_FOUND", Message: "not found"}
		err2 := &AppError{Code: "BAD_REQUEST", Message: "bad request"}

		assert.False(t, err1.Is(err2))
	})

	t.Run("matches wrapped sentinel error", func(t *testing.T) {
		err := &AppError{
			Code:    "NOT_FOUND",
			Message: "not found",
			Err:     ErrNotFound,
		}

		assert.True(t, err.Is(ErrNotFound))
	})
}

func TestErrorCheckers(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		assert.True(t, IsNotFound(ErrNotFound))
		assert.True(t, IsNotFound(NotFound("user")))
		assert.False(t, IsNotFound(ErrUnauthorized))
	})

	t.Run("IsUnauthorized", func(t *testing.T) {
		assert.True(t, IsUnauthorized(ErrUnauthorized))
		assert.True(t, IsUnauthorized(Unauthorized("invalid")))
		assert.False(t, IsUnauthorized(ErrNotFound))
	})

	t.Run("IsForbidden", func(t *testing.T) {
		assert.True(t, IsForbidden(ErrForbidden))
		assert.True(t, IsForbidden(Forbidden("denied")))
		assert.False(t, IsForbidden(ErrNotFound))
	})

	t.Run("IsConflict", func(t *testing.T) {
		assert.True(t, IsConflict(ErrConflict))
		assert.True(t, IsConflict(Conflict("exists")))
		assert.False(t, IsConflict(ErrNotFound))
	})

	t.Run("IsRateLimited", func(t *testing.T) {
		assert.True(t, IsRateLimited(ErrRateLimited))
		assert.True(t, IsRateLimited(RateLimited("slow down")))
		assert.False(t, IsRateLimited(ErrNotFound))
	})
}
