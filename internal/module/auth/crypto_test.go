package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCryptoManager(t *testing.T) {
	t.Run("creates with 32-byte key", func(t *testing.T) {
		key := "12345678901234567890123456789012" // Exactly 32 bytes
		manager, err := NewCryptoManager(key)
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("pads short key", func(t *testing.T) {
		key := "short-key"
		manager, err := NewCryptoManager(key)
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("truncates long key", func(t *testing.T) {
		key := "this-is-a-very-long-key-that-exceeds-32-bytes"
		manager, err := NewCryptoManager(key)
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})
}

func TestCryptoManager_EncryptDecrypt(t *testing.T) {
	key := "test-encryption-key-32-bytes!!"
	manager, err := NewCryptoManager(key)
	require.NoError(t, err)

	t.Run("encrypts and decrypts successfully", func(t *testing.T) {
		plaintext := "sk-test-api-key-12345"

		encrypted, err := manager.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := manager.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("produces different ciphertext each time", func(t *testing.T) {
		plaintext := "sk-test-api-key-12345"

		encrypted1, err := manager.Encrypt(plaintext)
		require.NoError(t, err)

		encrypted2, err := manager.Encrypt(plaintext)
		require.NoError(t, err)

		// Due to random nonce, ciphertext should be different
		assert.NotEqual(t, encrypted1, encrypted2)

		// But both should decrypt to same plaintext
		decrypted1, err := manager.Decrypt(encrypted1)
		require.NoError(t, err)

		decrypted2, err := manager.Decrypt(encrypted2)
		require.NoError(t, err)

		assert.Equal(t, decrypted1, decrypted2)
	})

	t.Run("handles empty string", func(t *testing.T) {
		encrypted, err := manager.Encrypt("")
		require.NoError(t, err)

		decrypted, err := manager.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, "", decrypted)
	})

	t.Run("handles unicode", func(t *testing.T) {
		plaintext := "测试密钥 🔐 こんにちは"

		encrypted, err := manager.Encrypt(plaintext)
		require.NoError(t, err)

		decrypted, err := manager.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("fails on invalid base64", func(t *testing.T) {
		_, err := manager.Decrypt("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("fails on tampered ciphertext", func(t *testing.T) {
		plaintext := "sk-test-api-key-12345"
		encrypted, err := manager.Encrypt(plaintext)
		require.NoError(t, err)

		// Tamper with the ciphertext
		tampered := encrypted[:len(encrypted)-4] + "XXXX"

		_, err = manager.Decrypt(tampered)
		assert.Error(t, err)
	})

	t.Run("fails with wrong key", func(t *testing.T) {
		plaintext := "sk-test-api-key-12345"
		encrypted, err := manager.Encrypt(plaintext)
		require.NoError(t, err)

		// Try to decrypt with different key
		otherManager, err := NewCryptoManager("different-key-32-bytes-long!!!")
		require.NoError(t, err)

		_, err = otherManager.Decrypt(encrypted)
		assert.Error(t, err)
	})
}

func TestGetKeyPrefix(t *testing.T) {
	t.Run("returns prefix with default length", func(t *testing.T) {
		prefix := GetKeyPrefix("sk-1234567890abcdef", 0)
		assert.Equal(t, "sk-1234...", prefix)
	})

	t.Run("returns prefix with custom length", func(t *testing.T) {
		prefix := GetKeyPrefix("sk-1234567890abcdef", 10)
		assert.Equal(t, "sk-1234567...", prefix)
	})

	t.Run("returns full key if shorter than length", func(t *testing.T) {
		prefix := GetKeyPrefix("sk-abc", 10)
		assert.Equal(t, "sk-abc", prefix)
	})

	t.Run("handles empty string", func(t *testing.T) {
		prefix := GetKeyPrefix("", 7)
		assert.Equal(t, "", prefix)
	})
}
