package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetUserIDFromContext extracts user ID from gin context.
// Returns the user ID and true if successful, or uuid.Nil and false if not found.
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}

	// Handle uuid.UUID type
	if userID, ok := userIDVal.(uuid.UUID); ok {
		return userID, true
	}

	// Handle string type
	if idStr, ok := userIDVal.(string); ok {
		userID, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
			return uuid.Nil, false
		}
		return userID, true
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
	return uuid.Nil, false
}

// MustGetUserID extracts user ID from context without error response.
// Returns uuid.Nil if not found or invalid.
func MustGetUserID(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}

	if userID, ok := userIDVal.(uuid.UUID); ok {
		return userID
	}

	if idStr, ok := userIDVal.(string); ok {
		userID, err := uuid.Parse(idStr)
		if err != nil {
			return uuid.Nil
		}
		return userID
	}

	return uuid.Nil
}
