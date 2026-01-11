package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/port/outbound"
)

const (
	// AuthorizationHeader is the header key for authorization.
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens.
	BearerPrefix = "Bearer "
	// UserIDKey is the context key for user ID.
	UserIDKey = "user_id"
	// EmailKey is the context key for email.
	EmailKey = "email"
)

// JWTValidator defines the interface for JWT token validation.
type JWTValidator interface {
	ValidateToken(token string) (*outbound.JWTClaims, error)
}

// Auth returns a middleware that validates JWT tokens.
// If the token is valid, it sets user_id and email in the context.
// If optional is true, the middleware will not abort on missing/invalid tokens.
func Auth(validator JWTValidator, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			if !optional {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "UNAUTHORIZED",
						"message": "Authorization header required",
					},
				})
			}
			c.Next()
			return
		}

		claims, err := validator.ValidateToken(token)
		if err != nil {
			if !optional {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "INVALID_TOKEN",
						"message": "Invalid or expired token",
					},
				})
			}
			c.Next()
			return
		}

		// Set user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)

		c.Next()
	}
}

// RequireAuth returns a middleware that requires a valid JWT token.
func RequireAuth(validator JWTValidator) gin.HandlerFunc {
	return Auth(validator, false)
}

// OptionalAuth returns a middleware that optionally validates JWT tokens.
func OptionalAuth(validator JWTValidator) gin.HandlerFunc {
	return Auth(validator, true)
}

// extractBearerToken extracts the bearer token from the Authorization header.
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return ""
	}

	if strings.HasPrefix(authHeader, BearerPrefix) {
		return strings.TrimPrefix(authHeader, BearerPrefix)
	}

	return ""
}

// GetUserID returns the user ID from context.
// Returns uuid.Nil if not found.
func GetUserID(c *gin.Context) uuid.UUID {
	if val, exists := c.Get(UserIDKey); exists {
		if userID, ok := val.(uuid.UUID); ok {
			return userID
		}
	}
	return uuid.Nil
}

// GetEmail returns the email from context.
// Returns empty string if not found.
func GetEmail(c *gin.Context) string {
	if val, exists := c.Get(EmailKey); exists {
		if email, ok := val.(string); ok {
			return email
		}
	}
	return ""
}

// IsAuthenticated returns true if the user is authenticated.
func IsAuthenticated(c *gin.Context) bool {
	return GetUserID(c) != uuid.Nil
}
