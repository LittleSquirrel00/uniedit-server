package random

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
)

// Charsets for different random string types.
const (
	CharsetAlphanumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	CharsetUpperAlphaNum = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	CharsetHex          = "0123456789abcdef"
)

// Hex generates a cryptographically secure random hex string.
// The output length is twice the input length (each byte = 2 hex chars).
func Hex(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// Base64URL generates a cryptographically secure random base64url string.
func Base64URL(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// String generates a random string from the given charset.
func String(length int, charset string) (string, error) {
	if length <= 0 {
		return "", nil
	}
	if charset == "" {
		charset = CharsetAlphanumeric
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("generate random index: %w", err)
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

// UpperAlphaNum generates a random uppercase alphanumeric string.
// Useful for order numbers, invoice numbers, etc.
func UpperAlphaNum(length int) string {
	s, _ := String(length, CharsetUpperAlphaNum)
	return s
}

// Token generates a cryptographically secure token (hex encoded).
// Useful for API keys, session tokens, etc.
func Token(length int) (string, error) {
	return Hex(length)
}

// SecureToken generates a URL-safe secure token.
// The actual length will be slightly longer due to base64 encoding.
func SecureToken(length int) (string, error) {
	s, err := Base64URL(length)
	if err != nil {
		return "", err
	}
	if len(s) > length {
		return s[:length], nil
	}
	return s, nil
}
