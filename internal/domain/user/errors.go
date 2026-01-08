package user

import "errors"

// Domain errors.
var (
	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrAccountDeleted     = errors.New("account deleted")
	ErrIncorrectPassword  = errors.New("incorrect password")
	ErrForbidden          = errors.New("forbidden")

	// Password errors
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordRequired = errors.New("password required for email users")

	// Verification errors
	ErrInvalidToken         = errors.New("invalid verification token")
	ErrTokenExpired         = errors.New("verification token expired")
	ErrTokenAlreadyUsed     = errors.New("verification token already used")
	ErrEmailAlreadyVerified = errors.New("email already verified")

	// Status errors
	ErrInvalidStatus      = errors.New("invalid user status")
	ErrCannotSuspendAdmin = errors.New("cannot suspend admin user")
	ErrAlreadySuspended   = errors.New("user is already suspended")
	ErrUserNotSuspended   = errors.New("user is not suspended")
	ErrAlreadyDeleted     = errors.New("user is already deleted")
)
