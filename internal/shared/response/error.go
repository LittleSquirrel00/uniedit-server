package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

// Error sends an error response with the given status code.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{Error: message})
}

// ErrorWithCode sends an error response with an error code.
func ErrorWithCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Error: message, Code: code})
}

// ErrorWithDetails sends an error response with additional details.
func ErrorWithDetails(c *gin.Context, status int, message string, details any) {
	c.JSON(status, ErrorResponse{Error: message, Details: details})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "forbidden"
	}
	Error(c, http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "not found"
	}
	Error(c, http.StatusNotFound, message)
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message)
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "internal error"
	}
	Error(c, http.StatusInternalServerError, message)
}

// ErrorMapping maps domain errors to HTTP status codes.
type ErrorMapping struct {
	Err     error
	Status  int
	Code    string
	Message string
}

// HandleError handles an error using the provided mappings.
// Returns true if the error was handled, false otherwise.
func HandleError(c *gin.Context, err error, mappings []ErrorMapping) bool {
	for _, m := range mappings {
		if errors.Is(err, m.Err) {
			msg := m.Message
			if msg == "" {
				msg = m.Err.Error()
			}
			if m.Code != "" {
				ErrorWithCode(c, m.Status, m.Code, msg)
			} else {
				Error(c, m.Status, msg)
			}
			return true
		}
	}
	return false
}

// HandleErrorWithDefault handles an error with a default fallback.
func HandleErrorWithDefault(c *gin.Context, err error, mappings []ErrorMapping) {
	if !HandleError(c, err, mappings) {
		InternalError(c, "")
	}
}
