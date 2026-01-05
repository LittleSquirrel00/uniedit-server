package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Common errors
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
	ErrBadRequest   = errors.New("bad request")
)

// getUserID extracts user ID from context.
func getUserID(c *gin.Context) (uuid.UUID, error) {
	// Try to get from auth middleware
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return id, nil
		}
		if idStr, ok := userID.(string); ok {
			return uuid.Parse(idStr)
		}
	}

	// Try to get from user object
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(interface{ GetID() uuid.UUID }); ok {
			return u.GetID(), nil
		}
	}

	return uuid.Nil, ErrUnauthorized
}

// handleError handles errors and returns appropriate HTTP response.
func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	errMsg := err.Error()

	// Check for common error patterns
	switch {
	case strings.Contains(errMsg, "not found"):
		c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
	case strings.Contains(errMsg, "unauthorized"):
		c.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
	case strings.Contains(errMsg, "invalid"):
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
	case strings.Contains(errMsg, "rate limit"):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": errMsg})
	case strings.Contains(errMsg, "timeout"):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": errMsg})
	case strings.Contains(errMsg, "unhealthy"):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errMsg})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// APIError represents an API error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// NewAPIError creates a new API error.
func NewAPIError(code, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to the error.
func (e *APIError) WithDetails(details any) *APIError {
	e.Details = details
	return e
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// GetOffset returns the offset for pagination.
func (p *Pagination) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the limit for pagination.
func (p *Pagination) GetLimit() int {
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.PageSize
}

// PagedResponse represents a paginated response.
type PagedResponse struct {
	Object     string `json:"object"`
	Data       any    `json:"data"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalCount int64  `json:"total_count"`
	TotalPages int    `json:"total_pages"`
}

// NewPagedResponse creates a new paged response.
func NewPagedResponse(data any, page, pageSize int, totalCount int64) *PagedResponse {
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	return &PagedResponse{
		Object:     "list",
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
