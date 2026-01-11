package middleware

import (
	"github.com/uniedit/server/internal/port/outbound"
)

// AuthDomainValidator wraps auth.AuthDomain to implement JWTValidator interface.
type AuthDomainValidator struct {
	validateFunc func(token string) (*outbound.JWTClaims, error)
}

// NewAuthDomainValidator creates a new AuthDomainValidator.
// The validateFunc should be auth.AuthDomain.ValidateAccessToken.
func NewAuthDomainValidator(validateFunc func(token string) (*outbound.JWTClaims, error)) *AuthDomainValidator {
	return &AuthDomainValidator{validateFunc: validateFunc}
}

// ValidateToken implements JWTValidator interface.
func (v *AuthDomainValidator) ValidateToken(token string) (*outbound.JWTClaims, error) {
	return v.validateFunc(token)
}

// Compile-time check
var _ JWTValidator = (*AuthDomainValidator)(nil)
