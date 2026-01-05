package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// CryptoManager handles encryption/decryption of sensitive data.
type CryptoManager struct {
	key []byte
}

// NewCryptoManager creates a new crypto manager with the given master key.
// The key must be 32 bytes (256 bits) for AES-256.
func NewCryptoManager(masterKey string) (*CryptoManager, error) {
	keyBytes := []byte(masterKey)

	// Ensure key is exactly 32 bytes for AES-256
	if len(keyBytes) < 32 {
		// Pad with zeros if too short (not recommended for production)
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	return &CryptoManager{key: keyBytes}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext (nonce + encrypted data + auth tag).
func (c *CryptoManager) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64 encoded
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM.
func (c *CryptoManager) Decrypt(encodedCiphertext string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrDecryptionFailed
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// GetKeyPrefix returns the first few characters of an API key for identification.
func GetKeyPrefix(apiKey string, length int) string {
	if length <= 0 {
		length = 7
	}
	if len(apiKey) <= length {
		return apiKey
	}
	return apiKey[:length] + "..."
}
