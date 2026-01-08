package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getUserID extracts the user ID from the Gin context.
// Returns uuid.Nil if not found or invalid.
func getUserID(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

// requireAuth checks if the user is authenticated and returns early if not.
// Returns the user ID if authenticated, uuid.Nil otherwise.
func requireAuth(c *gin.Context) uuid.UUID {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
		return uuid.Nil
	}
	return userID
}

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// respondError sends an error response with the given status code.
func respondError(c *gin.Context, statusCode int, errorCode string, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// respondSuccess sends a success response with the given data.
func respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// respondCreated sends a 201 Created response with the given data.
func respondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// PaginationParams represents common pagination parameters.
type PaginationParams struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GetPage returns the page number with a default of 1.
func (p *PaginationParams) GetPage() int {
	if p.Page <= 0 {
		return 1
	}
	return p.Page
}

// GetPageSize returns the page size with a default of 20.
func (p *PaginationParams) GetPageSize() int {
	if p.PageSize <= 0 {
		return 20
	}
	return p.PageSize
}

// GetOffset calculates the offset for database queries.
func (p *PaginationParams) GetOffset() int {
	return (p.GetPage() - 1) * p.GetPageSize()
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response.
func NewPaginatedResponse(data interface{}, total int64, page, pageSize int) PaginatedResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// respondPaginated sends a paginated response.
func respondPaginated(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, NewPaginatedResponse(data, total, page, pageSize))
}
