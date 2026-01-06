package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// APIKeyPrefix is the prefix for system API keys.
	APIKeyPrefix = "sk-"
	// APIKeyRandomLength is the length of the random part of the key.
	APIKeyRandomLength = 48
	// APIKeyPrefixDisplayLength is the length of the key prefix to display.
	APIKeyPrefixDisplayLength = 12
)

// GenerateAPIKey generates a new system API key.
// Returns the full key, its SHA-256 hash, and a display prefix.
func GenerateAPIKey() (key string, hash string, prefix string, err error) {
	// Generate random bytes
	randomBytes := make([]byte, APIKeyRandomLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Encode to hex and take first APIKeyRandomLength characters
	randomPart := hex.EncodeToString(randomBytes)[:APIKeyRandomLength]

	// Build the full key
	key = APIKeyPrefix + randomPart

	// Calculate hash
	hash = HashAPIKey(key)

	// Get display prefix
	prefix = GetAPIKeyPrefix(key)

	return key, hash, prefix, nil
}

// HashAPIKey returns the SHA-256 hash of an API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// GetAPIKeyPrefix returns the display prefix of an API key.
func GetAPIKeyPrefix(key string) string {
	if len(key) <= APIKeyPrefixDisplayLength {
		return key
	}
	return key[:APIKeyPrefixDisplayLength]
}

// IsValidAPIKeyFormat checks if a string looks like a valid API key format.
func IsValidAPIKeyFormat(key string) bool {
	if !strings.HasPrefix(key, APIKeyPrefix) {
		return false
	}
	// sk- + 48 hex characters = 51 total
	expectedLength := len(APIKeyPrefix) + APIKeyRandomLength
	return len(key) == expectedLength
}

// DefaultScopes returns the default scopes for a new API key.
func DefaultScopes() []string {
	return []string{
		string(APIKeyScopeChat),
		string(APIKeyScopeImage),
		string(APIKeyScopeEmbedding),
	}
}

// AllScopes returns all available scopes.
func AllScopes() []string {
	return []string{
		string(APIKeyScopeChat),
		string(APIKeyScopeImage),
		string(APIKeyScopeVideo),
		string(APIKeyScopeAudio),
		string(APIKeyScopeEmbedding),
	}
}

// ValidateScopes validates that all provided scopes are valid.
func ValidateScopes(scopes []string) error {
	validScopes := make(map[string]bool)
	for _, s := range AllScopes() {
		validScopes[s] = true
	}

	for _, scope := range scopes {
		if !validScopes[scope] {
			return fmt.Errorf("invalid scope: %s", scope)
		}
	}
	return nil
}
