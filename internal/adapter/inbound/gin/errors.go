package gin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
)

// handleError maps domain errors to HTTP responses.
func handleError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, user.ErrUserNotFound):
		statusCode = http.StatusNotFound
		errorCode = "user_not_found"
		message = "User not found"

	case errors.Is(err, user.ErrEmailAlreadyExists):
		statusCode = http.StatusConflict
		errorCode = "email_exists"
		message = "Email already registered"

	case errors.Is(err, user.ErrInvalidCredentials):
		statusCode = http.StatusUnauthorized
		errorCode = "invalid_credentials"
		message = "Invalid credentials"

	case errors.Is(err, user.ErrEmailNotVerified):
		statusCode = http.StatusForbidden
		errorCode = "email_not_verified"
		message = "Email not verified"

	case errors.Is(err, user.ErrAccountSuspended):
		statusCode = http.StatusForbidden
		errorCode = "account_suspended"
		message = "Account suspended"

	case errors.Is(err, user.ErrAccountDeleted):
		statusCode = http.StatusForbidden
		errorCode = "account_deleted"
		message = "Account deleted"

	case errors.Is(err, user.ErrIncorrectPassword):
		statusCode = http.StatusUnauthorized
		errorCode = "incorrect_password"
		message = "Incorrect password"

	case errors.Is(err, user.ErrForbidden):
		statusCode = http.StatusForbidden
		errorCode = "forbidden"
		message = "Forbidden"

	case errors.Is(err, user.ErrPasswordTooShort):
		statusCode = http.StatusBadRequest
		errorCode = "password_too_short"
		message = "Password must be at least 8 characters"

	case errors.Is(err, user.ErrPasswordRequired):
		statusCode = http.StatusBadRequest
		errorCode = "password_required"
		message = "Password required for email users"

	case errors.Is(err, user.ErrInvalidToken):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_token"
		message = "Invalid verification token"

	case errors.Is(err, user.ErrTokenExpired):
		statusCode = http.StatusBadRequest
		errorCode = "token_expired"
		message = "Verification token expired"

	case errors.Is(err, user.ErrTokenAlreadyUsed):
		statusCode = http.StatusBadRequest
		errorCode = "token_used"
		message = "Verification token already used"

	case errors.Is(err, user.ErrCannotSuspendAdmin):
		statusCode = http.StatusForbidden
		errorCode = "cannot_suspend_admin"
		message = "Cannot suspend admin user"

	case errors.Is(err, user.ErrUserAlreadyActive):
		statusCode = http.StatusBadRequest
		errorCode = "user_already_active"
		message = "User is already active"

	default:
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "Internal server error"
	}

	c.JSON(statusCode, model.ErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}
