package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Common error types.
var (
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrBadRequest    = errors.New("bad request")
	ErrConflict      = errors.New("resource conflict")
	ErrInternal      = errors.New("internal error")
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrRateLimited   = errors.New("rate limited")
)

// AppError represents an application error with HTTP status and error code.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// ErrorResponse represents the JSON error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewAppError creates a new application error.
func NewAppError(code string, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Common error constructors.

// NotFound creates a not found error.
func NotFound(resource string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
		Err:        ErrNotFound,
	}
}

// Unauthorized creates an unauthorized error.
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "authentication required"
	}
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Err:        ErrUnauthorized,
	}
}

// Forbidden creates a forbidden error.
func Forbidden(message string) *AppError {
	if message == "" {
		message = "access denied"
	}
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		StatusCode: http.StatusForbidden,
		Err:        ErrForbidden,
	}
}

// BadRequest creates a bad request error.
func BadRequest(message string) *AppError {
	return &AppError{
		Code:       "BAD_REQUEST",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        ErrBadRequest,
	}
}

// ValidationError creates a validation error.
func ValidationError(message string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		StatusCode: http.StatusUnprocessableEntity,
		Err:        ErrBadRequest,
	}
}

// Conflict creates a conflict error.
func Conflict(message string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    message,
		StatusCode: http.StatusConflict,
		Err:        ErrConflict,
	}
}

// Internal creates an internal error.
func Internal(message string, err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// QuotaExceeded creates a quota exceeded error.
func QuotaExceeded(message string) *AppError {
	return &AppError{
		Code:       "QUOTA_EXCEEDED",
		Message:    message,
		StatusCode: http.StatusPaymentRequired,
		Err:        ErrQuotaExceeded,
	}
}

// RateLimited creates a rate limited error.
func RateLimited(message string) *AppError {
	if message == "" {
		message = "too many requests"
	}
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
		Err:        ErrRateLimited,
	}
}

// ToResponse converts an AppError to ErrorResponse.
func (e *AppError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    e.Code,
			Message: e.Message,
		},
	}
}

// GetStatusCode returns the appropriate HTTP status code for an error.
func GetStatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrQuotaExceeded):
		return http.StatusPaymentRequired
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
