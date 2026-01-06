package pool

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Manager implements AccountPoolManager interface.
type Manager struct {
	repo      Repository
	scheduler Scheduler
	health    *HealthMonitor
	logger    *zap.Logger

	// Encryption key for API keys
	encryptionKey []byte

	// Cache for active accounts
	mu         sync.RWMutex
	accountsCache map[uuid.UUID][]*ProviderAccount
	cacheExpiry   map[uuid.UUID]time.Time
	cacheTTL      time.Duration
}

// ManagerConfig holds configuration for the manager.
type ManagerConfig struct {
	SchedulerType SchedulerType
	CacheTTL      time.Duration
	EncryptionKey string // 32-byte key for AES-256
}

// DefaultManagerConfig returns default configuration.
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		SchedulerType: SchedulerRoundRobin,
		CacheTTL:      5 * time.Minute,
	}
}

// NewManager creates a new account pool manager.
func NewManager(repo Repository, logger *zap.Logger, cfg *ManagerConfig) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultManagerConfig()
	}

	var encKey []byte
	if cfg.EncryptionKey != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decode encryption key: %w", err)
		}
		if len(key) != 32 {
			return nil, errors.New("encryption key must be 32 bytes")
		}
		encKey = key
	}

	return &Manager{
		repo:          repo,
		scheduler:     NewScheduler(cfg.SchedulerType),
		health:        NewHealthMonitor(),
		logger:        logger,
		encryptionKey: encKey,
		accountsCache: make(map[uuid.UUID][]*ProviderAccount),
		cacheExpiry:   make(map[uuid.UUID]time.Time),
		cacheTTL:      cfg.CacheTTL,
	}, nil
}

// GetAccount returns an available account for the provider.
func (m *Manager) GetAccount(ctx context.Context, providerID uuid.UUID) (*ProviderAccount, error) {
	accounts, err := m.getActiveAccounts(ctx, providerID)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, ErrNoAvailableAccount
	}

	// Filter to available accounts
	candidates := make([]*ProviderAccount, 0, len(accounts))
	for _, acc := range accounts {
		if acc.IsAvailable() && m.health.CanAttemptRequest(acc) {
			candidates = append(candidates, acc)
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoAvailableAccount
	}

	// Select account using scheduler
	selected, err := m.scheduler.Select(ctx, candidates)
	if err != nil {
		return nil, err
	}

	// Decrypt API key
	if m.encryptionKey != nil && selected.EncryptedAPIKey != "" {
		decrypted, err := m.decryptAPIKey(selected.EncryptedAPIKey)
		if err != nil {
			m.logger.Error("failed to decrypt API key",
				zap.String("account_id", selected.ID.String()),
				zap.Error(err))
			return nil, fmt.Errorf("decrypt API key: %w", err)
		}
		selected.DecryptedKey = decrypted
	}

	return selected, nil
}

// MarkSuccess records a successful request for the account.
func (m *Manager) MarkSuccess(ctx context.Context, accountID uuid.UUID, tokens int, costUSD float64) error {
	account, err := m.repo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}

	// Update health status
	newStatus, failures := m.health.RecordSuccess(account)
	if newStatus != account.HealthStatus {
		if err := m.repo.UpdateHealthStatus(ctx, accountID, newStatus, failures); err != nil {
			m.logger.Warn("failed to update health status", zap.Error(err))
		}
		m.invalidateCache(account.ProviderID)
	}

	// Record usage
	if err := m.repo.RecordSuccess(ctx, accountID, tokens, costUSD); err != nil {
		return err
	}

	// Record daily stats
	if err := m.repo.RecordDailyUsage(ctx, accountID, tokens, costUSD); err != nil {
		m.logger.Warn("failed to record daily usage", zap.Error(err))
	}

	return nil
}

// MarkFailure records a failed request for the account.
func (m *Manager) MarkFailure(ctx context.Context, accountID uuid.UUID, err error) error {
	account, accountErr := m.repo.GetByID(ctx, accountID)
	if accountErr != nil {
		return accountErr
	}

	// Update health status
	newStatus, failures := m.health.RecordFailure(account)
	if newStatus != account.HealthStatus || failures != account.ConsecutiveFailures {
		if updateErr := m.repo.UpdateHealthStatus(ctx, accountID, newStatus, failures); updateErr != nil {
			m.logger.Warn("failed to update health status", zap.Error(updateErr))
		}
		m.invalidateCache(account.ProviderID)
	}

	// Record failure
	if recordErr := m.repo.RecordFailure(ctx, accountID); recordErr != nil {
		return recordErr
	}

	m.logger.Warn("account request failed",
		zap.String("account_id", accountID.String()),
		zap.String("health_status", string(newStatus)),
		zap.Int("failures", failures),
		zap.Error(err))

	return nil
}

// MarkHighLatency records a high latency event.
func (m *Manager) MarkHighLatency(ctx context.Context, accountID uuid.UUID, latencyMs int) error {
	account, err := m.repo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}

	newStatus, failures := m.health.RecordHighLatency(account, latencyMs)
	if newStatus != account.HealthStatus {
		if updateErr := m.repo.UpdateHealthStatus(ctx, accountID, newStatus, failures); updateErr != nil {
			m.logger.Warn("failed to update health status", zap.Error(updateErr))
		}
		m.invalidateCache(account.ProviderID)
	}

	return nil
}

// GetAccountStats returns usage statistics for an account.
func (m *Manager) GetAccountStats(ctx context.Context, accountID uuid.UUID) (*AccountStats, error) {
	return m.repo.GetStats(ctx, accountID, 30)
}

// RefreshHealth triggers health checks for all accounts of a provider.
func (m *Manager) RefreshHealth(ctx context.Context, providerID uuid.UUID) error {
	m.invalidateCache(providerID)
	return nil
}

// AddAccount adds a new account to the pool.
func (m *Manager) AddAccount(ctx context.Context, providerID uuid.UUID, name, apiKey string, weight, priority int) (*ProviderAccount, error) {
	// Encrypt API key
	var encryptedKey string
	var keyPrefix string
	if m.encryptionKey != nil {
		encrypted, err := m.encryptAPIKey(apiKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt API key: %w", err)
		}
		encryptedKey = encrypted
	} else {
		encryptedKey = apiKey // Store as-is if no encryption key
	}

	// Extract prefix for identification
	if len(apiKey) > 8 {
		keyPrefix = apiKey[:8] + "..."
	} else {
		keyPrefix = apiKey
	}

	account := &ProviderAccount{
		ProviderID:      providerID,
		Name:            name,
		EncryptedAPIKey: encryptedKey,
		KeyPrefix:       keyPrefix,
		Weight:          weight,
		Priority:        priority,
		IsActive:        true,
		HealthStatus:    HealthStatusHealthy,
	}

	if err := m.repo.Create(ctx, account); err != nil {
		return nil, err
	}

	m.invalidateCache(providerID)
	return account, nil
}

// UpdateAccount updates an account configuration.
func (m *Manager) UpdateAccount(ctx context.Context, account *ProviderAccount) error {
	if err := m.repo.Update(ctx, account); err != nil {
		return err
	}
	m.invalidateCache(account.ProviderID)
	return nil
}

// RemoveAccount removes an account from the pool.
func (m *Manager) RemoveAccount(ctx context.Context, accountID uuid.UUID) error {
	account, err := m.repo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}

	if err := m.repo.Delete(ctx, accountID); err != nil {
		return err
	}

	m.invalidateCache(account.ProviderID)
	m.health.Reset(accountID.String())
	return nil
}

// ListAccounts returns all accounts for a provider.
func (m *Manager) ListAccounts(ctx context.Context, providerID uuid.UUID) ([]*ProviderAccount, error) {
	return m.repo.GetAllByProvider(ctx, providerID)
}

// getActiveAccounts returns active accounts, using cache if available.
func (m *Manager) getActiveAccounts(ctx context.Context, providerID uuid.UUID) ([]*ProviderAccount, error) {
	m.mu.RLock()
	if accounts, ok := m.accountsCache[providerID]; ok {
		if expiry, exists := m.cacheExpiry[providerID]; exists && time.Now().Before(expiry) {
			m.mu.RUnlock()
			return accounts, nil
		}
	}
	m.mu.RUnlock()

	// Fetch from database
	accounts, err := m.repo.GetActiveByProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// Update cache
	m.mu.Lock()
	m.accountsCache[providerID] = accounts
	m.cacheExpiry[providerID] = time.Now().Add(m.cacheTTL)
	m.mu.Unlock()

	return accounts, nil
}

// invalidateCache invalidates the cache for a provider.
func (m *Manager) invalidateCache(providerID uuid.UUID) {
	m.mu.Lock()
	delete(m.accountsCache, providerID)
	delete(m.cacheExpiry, providerID)
	m.mu.Unlock()
}

// encryptAPIKey encrypts an API key using AES-256-GCM.
func (m *Manager) encryptAPIKey(plaintext string) (string, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptAPIKey decrypts an API key using AES-256-GCM.
func (m *Manager) decryptAPIKey(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
