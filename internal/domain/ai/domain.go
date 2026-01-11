package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// AIDomain defines the AI domain service interface.
type AIDomain interface {
	// Chat sends a non-streaming chat completion request.
	Chat(ctx context.Context, userID uuid.UUID, req *model.AIChatRequest) (*model.AIChatResponse, error)

	// ChatStream sends a streaming chat completion request.
	ChatStream(ctx context.Context, userID uuid.UUID, req *model.AIChatRequest) (<-chan *model.AIChatChunk, *model.AIRoutingInfo, error)

	// Embed generates text embeddings.
	Embed(ctx context.Context, userID uuid.UUID, req *model.AIEmbedRequest) (*model.AIEmbedResponse, error)

	// Route performs routing decision (for testing/debugging).
	Route(ctx context.Context, routingCtx *model.AIRoutingContext) (*model.AIRoutingResult, error)

	// Provider management
	GetProvider(ctx context.Context, id uuid.UUID) (*model.AIProvider, error)
	ListProviders(ctx context.Context) ([]*model.AIProvider, error)
	CreateProvider(ctx context.Context, provider *model.AIProvider) error
	UpdateProvider(ctx context.Context, provider *model.AIProvider) error
	DeleteProvider(ctx context.Context, id uuid.UUID) error

	// Model management
	GetModel(ctx context.Context, id string) (*model.AIModel, error)
	ListModels(ctx context.Context) ([]*model.AIModel, error)
	ListModelsByCapability(ctx context.Context, cap model.AICapability) ([]*model.AIModel, error)
	CreateModel(ctx context.Context, m *model.AIModel) error
	UpdateModel(ctx context.Context, m *model.AIModel) error
	DeleteModel(ctx context.Context, id string) error

	// Account pool management
	GetAccount(ctx context.Context, id uuid.UUID) (*model.AIProviderAccount, error)
	ListAccounts(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error)
	ListAccountsByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error)
	CreateAccount(ctx context.Context, account *model.AIProviderAccount, apiKey string) error
	UpdateAccount(ctx context.Context, account *model.AIProviderAccount) error
	DeleteAccount(ctx context.Context, id uuid.UUID) error
	GetAccountStats(ctx context.Context, id uuid.UUID, days int) ([]*model.AIAccountUsageStats, error)
	ResetAccountHealth(ctx context.Context, id uuid.UUID) error

	// Group management
	GetGroup(ctx context.Context, id string) (*model.AIModelGroup, error)
	ListGroups(ctx context.Context) ([]*model.AIModelGroup, error)
	CreateGroup(ctx context.Context, group *model.AIModelGroup) error
	UpdateGroup(ctx context.Context, group *model.AIModelGroup) error
	DeleteGroup(ctx context.Context, id string) error

	// Public API
	ListEnabledModels(ctx context.Context) ([]*model.AIModel, error)

	// Provider operations
	SyncModels(ctx context.Context, providerID uuid.UUID) error
	ProviderHealthCheck(ctx context.Context, providerID uuid.UUID) (bool, error)

	// Health monitoring
	StartHealthMonitor(ctx context.Context)
	StopHealthMonitor()
	IsProviderHealthy(providerID uuid.UUID) bool
	IsAccountHealthy(accountID uuid.UUID) bool
}

// aiDomain implements AIDomain.
type aiDomain struct {
	// Database ports
	providerDB outbound.AIProviderDatabasePort
	modelDB    outbound.AIModelDatabasePort
	accountDB  outbound.AIProviderAccountDatabasePort
	groupDB    outbound.AIModelGroupDatabasePort

	// Cache ports
	healthCache    outbound.AIProviderHealthCachePort
	embeddingCache outbound.AIEmbeddingCachePort

	// Adapter ports
	vendorRegistry outbound.AIVendorRegistryPort
	crypto         outbound.AICryptoPort
	usageRecorder  outbound.AIUsageRecorderPort

	// Routing
	strategyChain *StrategyChain

	// In-memory caches (for fast routing)
	providerCache   map[uuid.UUID]*model.AIProvider
	modelCache      map[string]*model.AIModel
	providerMu      sync.RWMutex

	// Health monitoring
	healthStatus    map[uuid.UUID]bool
	accountHealth   map[uuid.UUID]model.AIHealthStatus
	healthMu        sync.RWMutex
	healthCtx       context.Context
	healthCancel    context.CancelFunc
	healthInterval  time.Duration

	logger *zap.Logger
}

// Config holds AI domain configuration.
type Config struct {
	HealthCheckInterval time.Duration
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		HealthCheckInterval: 30 * time.Second,
	}
}

// NewAIDomain creates a new AI domain service.
func NewAIDomain(
	providerDB outbound.AIProviderDatabasePort,
	modelDB outbound.AIModelDatabasePort,
	accountDB outbound.AIProviderAccountDatabasePort,
	groupDB outbound.AIModelGroupDatabasePort,
	healthCache outbound.AIProviderHealthCachePort,
	embeddingCache outbound.AIEmbeddingCachePort,
	vendorRegistry outbound.AIVendorRegistryPort,
	crypto outbound.AICryptoPort,
	usageRecorder outbound.AIUsageRecorderPort,
	config *Config,
	logger *zap.Logger,
) AIDomain {
	if config == nil {
		config = DefaultConfig()
	}

	d := &aiDomain{
		providerDB:     providerDB,
		modelDB:        modelDB,
		accountDB:      accountDB,
		groupDB:        groupDB,
		healthCache:    healthCache,
		embeddingCache: embeddingCache,
		vendorRegistry: vendorRegistry,
		crypto:         crypto,
		usageRecorder:  usageRecorder,
		strategyChain:  DefaultStrategyChain(),
		providerCache:  make(map[uuid.UUID]*model.AIProvider),
		modelCache:     make(map[string]*model.AIModel),
		healthStatus:   make(map[uuid.UUID]bool),
		accountHealth:  make(map[uuid.UUID]model.AIHealthStatus),
		healthInterval: config.HealthCheckInterval,
		logger:         logger,
	}

	return d
}

// ===== Chat Operations =====

// Chat performs a non-streaming chat completion.
func (d *aiDomain) Chat(ctx context.Context, userID uuid.UUID, req *model.AIChatRequest) (*model.AIChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, ErrEmptyMessages
	}

	startTime := time.Now()

	// Build routing context
	routingCtx := d.buildRoutingContext(req)

	// Route to best model
	result, err := d.Route(ctx, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get adapter
	adapter, err := d.vendorRegistry.GetForProvider(result.Provider)
	if err != nil {
		return nil, fmt.Errorf("get adapter: %w", err)
	}

	// Build adapter request
	adapterReq := &model.AIChatRequest{
		Model:       result.Model.ID,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Tools:       req.Tools,
		ToolChoice:  req.ToolChoice,
		Stream:      false,
		Metadata:    req.Metadata,
	}

	// Execute request
	resp, err := adapter.Chat(ctx, adapterReq, result.Model, result.Provider, result.APIKey)
	if err != nil {
		// Mark failure for health tracking
		d.markRequestFailure(ctx, result, err)
		return nil, fmt.Errorf("chat failed: %w", err)
	}

	// Calculate latency and cost
	latencyMs := time.Since(startTime).Milliseconds()
	costUSD := d.calculateCost(result.Model, resp.Usage)

	// Mark success
	d.markRequestSuccess(ctx, result, resp.Usage, costUSD)

	// Record usage for billing
	if d.usageRecorder != nil && resp.Usage != nil {
		_ = d.usageRecorder.RecordUsage(ctx, userID, result.Model.ID, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, costUSD)
	}

	// Add routing info
	resp.Routing = &model.AIRoutingInfo{
		ProviderUsed: result.Provider.Name,
		ModelUsed:    result.Model.ID,
		LatencyMs:    latencyMs,
		CostUSD:      costUSD,
	}

	return resp, nil
}

// ChatStream performs a streaming chat completion.
func (d *aiDomain) ChatStream(ctx context.Context, userID uuid.UUID, req *model.AIChatRequest) (<-chan *model.AIChatChunk, *model.AIRoutingInfo, error) {
	if len(req.Messages) == 0 {
		return nil, nil, ErrEmptyMessages
	}

	// Build routing context
	routingCtx := d.buildRoutingContext(req)
	routingCtx.RequireStream = true

	// Route to best model
	result, err := d.Route(ctx, routingCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get adapter
	adapter, err := d.vendorRegistry.GetForProvider(result.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("get adapter: %w", err)
	}

	// Build adapter request
	adapterReq := &model.AIChatRequest{
		Model:       result.Model.ID,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Tools:       req.Tools,
		ToolChoice:  req.ToolChoice,
		Stream:      true,
		Metadata:    req.Metadata,
	}

	// Execute streaming request
	chunks, err := adapter.ChatStream(ctx, adapterReq, result.Model, result.Provider, result.APIKey)
	if err != nil {
		d.markRequestFailure(ctx, result, err)
		return nil, nil, fmt.Errorf("chat stream failed: %w", err)
	}

	routingInfo := &model.AIRoutingInfo{
		ProviderUsed: result.Provider.Name,
		ModelUsed:    result.Model.ID,
	}

	return chunks, routingInfo, nil
}

// Embed generates text embeddings.
func (d *aiDomain) Embed(ctx context.Context, userID uuid.UUID, req *model.AIEmbedRequest) (*model.AIEmbedResponse, error) {
	if len(req.Input) == 0 {
		return nil, ErrEmptyInput
	}

	// Build routing context for embedding
	routingCtx := model.NewAIRoutingContext()
	routingCtx.TaskType = string(model.AITaskTypeEmbedding)

	// If specific model requested, try to use it
	if req.Model != "" && req.Model != "auto" {
		routingCtx.PreferredModels = []string{req.Model}
	}

	// Route to best model
	result, err := d.Route(ctx, routingCtx)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get adapter
	adapter, err := d.vendorRegistry.GetForProvider(result.Provider)
	if err != nil {
		return nil, fmt.Errorf("get adapter: %w", err)
	}

	// Execute request
	resp, err := adapter.Embed(ctx, req, result.Model, result.Provider, result.APIKey)
	if err != nil {
		d.markRequestFailure(ctx, result, err)
		return nil, fmt.Errorf("embed failed: %w", err)
	}

	// Mark success
	if resp.Usage != nil {
		costUSD := d.calculateCost(result.Model, resp.Usage)
		d.markRequestSuccess(ctx, result, resp.Usage, costUSD)

		if d.usageRecorder != nil {
			_ = d.usageRecorder.RecordUsage(ctx, userID, result.Model.ID, resp.Usage.PromptTokens, 0, costUSD)
		}
	}

	return resp, nil
}

// ===== Routing =====

// Route performs routing decision.
func (d *aiDomain) Route(ctx context.Context, routingCtx *model.AIRoutingContext) (*model.AIRoutingResult, error) {
	// Get candidates
	candidates, err := d.getCandidates(ctx, routingCtx)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, ErrNoAvailableModels
	}

	// Inject health status
	d.healthMu.RLock()
	for providerID, healthy := range d.healthStatus {
		routingCtx.ProviderHealth[providerID.String()] = healthy
	}
	d.healthMu.RUnlock()

	// Execute strategy chain
	result, err := d.strategyChain.Execute(routingCtx, candidates)
	if err != nil {
		return nil, err
	}

	// Resolve API key
	if err := d.resolveAPIKey(ctx, result); err != nil {
		return nil, fmt.Errorf("resolve API key: %w", err)
	}

	return result, nil
}

// getCandidates builds the list of candidate models.
func (d *aiDomain) getCandidates(ctx context.Context, routingCtx *model.AIRoutingContext) ([]*model.AIScoredCandidate, error) {
	var models []*model.AIModel
	var err error

	// If group is specified, use group models
	if routingCtx.GroupID != "" {
		group, err := d.groupDB.FindByID(ctx, routingCtx.GroupID)
		if err != nil {
			return nil, fmt.Errorf("get group: %w", err)
		}
		if group == nil {
			return nil, ErrGroupNotFound
		}

		for _, modelID := range group.Models {
			if m, err := d.modelDB.FindByID(ctx, modelID); err == nil && m != nil && m.Enabled {
				models = append(models, m)
			}
		}
	} else {
		// Get all enabled models with required capabilities
		caps := routingCtx.RequiredCapabilities()
		if len(caps) > 0 {
			models, err = d.modelDB.FindByCapabilities(ctx, caps)
		} else {
			models, err = d.modelDB.FindEnabled(ctx)
		}
		if err != nil {
			return nil, err
		}
	}

	// Build scored candidates
	candidates := make([]*model.AIScoredCandidate, 0, len(models))
	for _, m := range models {
		if !m.Enabled {
			continue
		}

		provider, err := d.providerDB.FindByID(ctx, m.ProviderID)
		if err != nil || provider == nil || !provider.Enabled {
			continue
		}

		candidates = append(candidates, model.NewAIScoredCandidate(provider, m))
	}

	return candidates, nil
}

// resolveAPIKey gets the API key from account pool or provider.
func (d *aiDomain) resolveAPIKey(ctx context.Context, result *model.AIRoutingResult) error {
	// Try to get account from pool
	if d.accountDB != nil {
		accounts, err := d.accountDB.FindAvailableByProvider(ctx, result.Provider.ID)
		if err == nil && len(accounts) > 0 {
			// Select best account (weighted random or priority)
			account := d.selectAccount(accounts)
			if account != nil {
				// Decrypt API key
				if d.crypto != nil {
					decrypted, err := d.crypto.Decrypt(account.EncryptedAPIKey)
					if err != nil {
						d.logger.Warn("failed to decrypt account API key",
							zap.String("account_id", account.ID.String()),
							zap.Error(err))
					} else {
						accountID := account.ID.String()
						result.AccountID = &accountID
						result.APIKey = decrypted
						return nil
					}
				}
			}
		}
	}

	// Fall back to provider's API key
	result.APIKey = result.Provider.APIKey
	return nil
}

// selectAccount selects the best account from available accounts.
func (d *aiDomain) selectAccount(accounts []*model.AIProviderAccount) *model.AIProviderAccount {
	if len(accounts) == 0 {
		return nil
	}

	// Simple priority-based selection
	var best *model.AIProviderAccount
	for _, acc := range accounts {
		if best == nil || acc.Priority > best.Priority {
			best = acc
		}
	}

	return best
}

// buildRoutingContext builds a routing context from a chat request.
func (d *aiDomain) buildRoutingContext(req *model.AIChatRequest) *model.AIRoutingContext {
	ctx := model.NewAIRoutingContext()
	ctx.TaskType = string(model.AITaskTypeChat)
	ctx.RequireStream = req.Stream

	// Detect required capabilities from messages
	for _, msg := range req.Messages {
		if msg.HasImages() {
			ctx.RequireVision = true
			break
		}
	}

	if len(req.Tools) > 0 {
		ctx.RequireTools = true
	}

	// If specific model requested
	if req.Model != "" && req.Model != "auto" {
		ctx.PreferredModels = []string{req.Model}
	}

	return ctx
}

// ===== Provider Management =====

func (d *aiDomain) GetProvider(ctx context.Context, id uuid.UUID) (*model.AIProvider, error) {
	return d.providerDB.FindByID(ctx, id)
}

func (d *aiDomain) ListProviders(ctx context.Context) ([]*model.AIProvider, error) {
	return d.providerDB.FindAll(ctx)
}

func (d *aiDomain) CreateProvider(ctx context.Context, provider *model.AIProvider) error {
	if provider.ID == uuid.Nil {
		provider.ID = uuid.New()
	}
	return d.providerDB.Create(ctx, provider)
}

func (d *aiDomain) UpdateProvider(ctx context.Context, provider *model.AIProvider) error {
	return d.providerDB.Update(ctx, provider)
}

func (d *aiDomain) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	// Delete associated models first
	if err := d.modelDB.DeleteByProvider(ctx, id); err != nil {
		return err
	}
	// Delete associated accounts
	if d.accountDB != nil {
		if err := d.accountDB.DeleteByProvider(ctx, id); err != nil {
			return err
		}
	}
	return d.providerDB.Delete(ctx, id)
}

// ===== Model Management =====

func (d *aiDomain) GetModel(ctx context.Context, id string) (*model.AIModel, error) {
	return d.modelDB.FindByID(ctx, id)
}

func (d *aiDomain) ListModels(ctx context.Context) ([]*model.AIModel, error) {
	return d.modelDB.FindEnabled(ctx)
}

func (d *aiDomain) ListModelsByCapability(ctx context.Context, cap model.AICapability) ([]*model.AIModel, error) {
	return d.modelDB.FindByCapability(ctx, cap)
}

func (d *aiDomain) CreateModel(ctx context.Context, m *model.AIModel) error {
	return d.modelDB.Create(ctx, m)
}

func (d *aiDomain) UpdateModel(ctx context.Context, m *model.AIModel) error {
	return d.modelDB.Update(ctx, m)
}

func (d *aiDomain) DeleteModel(ctx context.Context, id string) error {
	return d.modelDB.Delete(ctx, id)
}

// ===== Account Pool Management =====

func (d *aiDomain) GetAccount(ctx context.Context, id uuid.UUID) (*model.AIProviderAccount, error) {
	if d.accountDB == nil {
		return nil, ErrAccountNotFound
	}
	return d.accountDB.FindByID(ctx, id)
}

func (d *aiDomain) ListAccounts(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	if d.accountDB == nil {
		return nil, nil
	}
	return d.accountDB.FindByProvider(ctx, providerID)
}

func (d *aiDomain) ListAccountsByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error) {
	return d.ListAccounts(ctx, providerID)
}

func (d *aiDomain) CreateAccount(ctx context.Context, account *model.AIProviderAccount, apiKey string) error {
	if d.accountDB == nil {
		return ErrAdapterNotFound
	}
	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}

	// Encrypt the API key if crypto is available
	if d.crypto != nil && apiKey != "" {
		encrypted, err := d.crypto.Encrypt(apiKey)
		if err != nil {
			return fmt.Errorf("encrypt API key: %w", err)
		}
		account.EncryptedAPIKey = encrypted
	}

	return d.accountDB.Create(ctx, account)
}

func (d *aiDomain) GetAccountStats(ctx context.Context, id uuid.UUID, days int) ([]*model.AIAccountUsageStats, error) {
	if d.accountDB == nil {
		return nil, ErrAdapterNotFound
	}
	// This would require an AIAccountUsageStatsDatabasePort - for now return empty
	return nil, nil
}

func (d *aiDomain) UpdateAccount(ctx context.Context, account *model.AIProviderAccount) error {
	if d.accountDB == nil {
		return ErrAdapterNotFound
	}
	return d.accountDB.Update(ctx, account)
}

func (d *aiDomain) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	if d.accountDB == nil {
		return ErrAdapterNotFound
	}
	return d.accountDB.Delete(ctx, id)
}

func (d *aiDomain) ResetAccountHealth(ctx context.Context, id uuid.UUID) error {
	if d.accountDB == nil {
		return ErrAdapterNotFound
	}
	return d.accountDB.UpdateHealth(ctx, id, model.AIHealthStatusHealthy, 0)
}

// ===== Group Management =====

func (d *aiDomain) GetGroup(ctx context.Context, id string) (*model.AIModelGroup, error) {
	return d.groupDB.FindByID(ctx, id)
}

func (d *aiDomain) ListGroups(ctx context.Context) ([]*model.AIModelGroup, error) {
	return d.groupDB.FindAll(ctx)
}

func (d *aiDomain) CreateGroup(ctx context.Context, group *model.AIModelGroup) error {
	return d.groupDB.Create(ctx, group)
}

func (d *aiDomain) UpdateGroup(ctx context.Context, group *model.AIModelGroup) error {
	return d.groupDB.Update(ctx, group)
}

func (d *aiDomain) DeleteGroup(ctx context.Context, id string) error {
	return d.groupDB.Delete(ctx, id)
}

// ===== Public API =====

func (d *aiDomain) ListEnabledModels(ctx context.Context) ([]*model.AIModel, error) {
	return d.modelDB.FindEnabled(ctx)
}

// ===== Provider Operations =====

func (d *aiDomain) SyncModels(ctx context.Context, providerID uuid.UUID) error {
	// This would sync models from the provider's API
	// For now, just return nil as models are managed manually
	return nil
}

func (d *aiDomain) ProviderHealthCheck(ctx context.Context, providerID uuid.UUID) (bool, error) {
	provider, err := d.providerDB.FindByID(ctx, providerID)
	if err != nil {
		return false, err
	}
	if provider == nil {
		return false, ErrProviderNotFound
	}

	adapter, err := d.vendorRegistry.GetForProvider(provider)
	if err != nil {
		return false, err
	}

	err = adapter.HealthCheck(ctx, provider, provider.APIKey)
	healthy := err == nil

	d.updateProviderHealth(providerID, healthy)

	return healthy, nil
}

// ===== Health Monitoring =====

// StartHealthMonitor starts background health monitoring.
func (d *aiDomain) StartHealthMonitor(ctx context.Context) {
	d.healthCtx, d.healthCancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(d.healthInterval)
		defer ticker.Stop()

		// Initial check
		d.runHealthCheck(d.healthCtx)

		for {
			select {
			case <-d.healthCtx.Done():
				return
			case <-ticker.C:
				d.runHealthCheck(d.healthCtx)
			}
		}
	}()

	d.logger.Info("health monitor started", zap.Duration("interval", d.healthInterval))
}

// StopHealthMonitor stops health monitoring.
func (d *aiDomain) StopHealthMonitor() {
	if d.healthCancel != nil {
		d.healthCancel()
	}
	d.logger.Info("health monitor stopped")
}

// runHealthCheck performs health checks on all providers.
func (d *aiDomain) runHealthCheck(ctx context.Context) {
	providers, err := d.providerDB.FindEnabled(ctx)
	if err != nil {
		d.logger.Error("failed to load providers for health check", zap.Error(err))
		return
	}

	for _, provider := range providers {
		adapter, err := d.vendorRegistry.GetForProvider(provider)
		if err != nil {
			d.updateProviderHealth(provider.ID, false)
			continue
		}

		err = adapter.HealthCheck(ctx, provider, provider.APIKey)
		healthy := err == nil

		d.updateProviderHealth(provider.ID, healthy)

		if !healthy {
			d.logger.Warn("provider health check failed",
				zap.String("provider", provider.Name),
				zap.Error(err))
		}
	}
}

// updateProviderHealth updates provider health status.
func (d *aiDomain) updateProviderHealth(providerID uuid.UUID, healthy bool) {
	d.healthMu.Lock()
	d.healthStatus[providerID] = healthy
	d.healthMu.Unlock()

	// Update cache if available
	if d.healthCache != nil {
		_ = d.healthCache.SetProviderHealth(context.Background(), providerID, healthy, d.healthInterval*2)
	}
}

// IsProviderHealthy returns provider health status.
func (d *aiDomain) IsProviderHealthy(providerID uuid.UUID) bool {
	d.healthMu.RLock()
	defer d.healthMu.RUnlock()
	healthy, ok := d.healthStatus[providerID]
	return ok && healthy
}

// IsAccountHealthy returns account health status.
func (d *aiDomain) IsAccountHealthy(accountID uuid.UUID) bool {
	d.healthMu.RLock()
	defer d.healthMu.RUnlock()
	status, ok := d.accountHealth[accountID]
	return ok && status.CanServeRequests()
}

// ===== Usage Tracking =====

// markRequestSuccess records a successful request.
func (d *aiDomain) markRequestSuccess(ctx context.Context, result *model.AIRoutingResult, usage *model.AIUsage, costUSD float64) {
	if result.AccountID == nil || d.accountDB == nil {
		return
	}

	accountID, err := uuid.Parse(*result.AccountID)
	if err != nil {
		return
	}

	tokens := int64(0)
	if usage != nil {
		tokens = int64(usage.TotalTokens)
	}

	_ = d.accountDB.IncrementUsage(ctx, accountID, 1, tokens, costUSD)

	// Update health status
	d.healthMu.Lock()
	d.accountHealth[accountID] = model.AIHealthStatusHealthy
	d.healthMu.Unlock()
}

// markRequestFailure records a failed request.
func (d *aiDomain) markRequestFailure(ctx context.Context, result *model.AIRoutingResult, err error) {
	if result.AccountID == nil || d.accountDB == nil {
		return
	}

	accountID, parseErr := uuid.Parse(*result.AccountID)
	if parseErr != nil {
		return
	}

	// Get current account
	account, findErr := d.accountDB.FindByID(ctx, accountID)
	if findErr != nil || account == nil {
		return
	}

	// Update failure count
	failures := account.ConsecutiveFailures + 1
	newStatus := account.HealthStatus

	if failures >= model.AIFailuresToUnhealthy {
		newStatus = model.AIHealthStatusUnhealthy
	} else if failures >= model.AIFailuresToDegrade {
		newStatus = model.AIHealthStatusDegraded
	}

	_ = d.accountDB.UpdateHealth(ctx, accountID, newStatus, failures)

	// Update in-memory cache
	d.healthMu.Lock()
	d.accountHealth[accountID] = newStatus
	d.healthMu.Unlock()

	d.logger.Warn("request failed",
		zap.String("account_id", accountID.String()),
		zap.Int("consecutive_failures", failures),
		zap.String("health_status", string(newStatus)),
		zap.Error(err))
}

// calculateCost calculates the cost of a request.
func (d *aiDomain) calculateCost(m *model.AIModel, usage *model.AIUsage) float64 {
	if usage == nil {
		return 0
	}

	inputCost := float64(usage.PromptTokens) / 1000 * m.InputCostPer1K
	outputCost := float64(usage.CompletionTokens) / 1000 * m.OutputCostPer1K

	return inputCost + outputCost
}
