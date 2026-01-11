package inbound

import "github.com/gin-gonic/gin"

// ===== Chat HTTP Ports =====

// AIChatHttpPort defines chat HTTP handler interface.
type AIChatHttpPort interface {
	// Chat handles POST /v1/chat/completions (non-streaming).
	Chat(c *gin.Context)

	// ChatStream handles POST /v1/chat/completions (streaming).
	ChatStream(c *gin.Context)
}

// ===== Embedding HTTP Ports =====

// AIEmbeddingHttpPort defines embedding HTTP handler interface.
type AIEmbeddingHttpPort interface {
	// Embed handles POST /v1/embeddings.
	Embed(c *gin.Context)
}

// ===== Provider Admin HTTP Ports =====

// AIProviderAdminHttpPort defines provider admin HTTP handler interface.
type AIProviderAdminHttpPort interface {
	// ListProviders handles GET /admin/ai/providers.
	ListProviders(c *gin.Context)

	// GetProvider handles GET /admin/ai/providers/:id.
	GetProvider(c *gin.Context)

	// CreateProvider handles POST /admin/ai/providers.
	CreateProvider(c *gin.Context)

	// UpdateProvider handles PUT /admin/ai/providers/:id.
	UpdateProvider(c *gin.Context)

	// DeleteProvider handles DELETE /admin/ai/providers/:id.
	DeleteProvider(c *gin.Context)

	// SyncModels handles POST /admin/ai/providers/:id/sync.
	SyncModels(c *gin.Context)

	// HealthCheck handles POST /admin/ai/providers/:id/health.
	HealthCheck(c *gin.Context)
}

// ===== Model Admin HTTP Ports =====

// AIModelAdminHttpPort defines model admin HTTP handler interface.
type AIModelAdminHttpPort interface {
	// ListModels handles GET /admin/ai/models.
	ListModels(c *gin.Context)

	// GetModel handles GET /admin/ai/models/:id.
	GetModel(c *gin.Context)

	// CreateModel handles POST /admin/ai/models.
	CreateModel(c *gin.Context)

	// UpdateModel handles PUT /admin/ai/models/:id.
	UpdateModel(c *gin.Context)

	// DeleteModel handles DELETE /admin/ai/models/:id.
	DeleteModel(c *gin.Context)
}

// ===== Account Pool HTTP Ports =====

// AIAccountPoolHttpPort defines account pool HTTP handler interface.
type AIAccountPoolHttpPort interface {
	// ListAccounts handles GET /admin/ai/providers/:provider_id/accounts.
	ListAccounts(c *gin.Context)

	// GetAccount handles GET /admin/ai/accounts/:id.
	GetAccount(c *gin.Context)

	// CreateAccount handles POST /admin/ai/providers/:provider_id/accounts.
	CreateAccount(c *gin.Context)

	// UpdateAccount handles PUT /admin/ai/accounts/:id.
	UpdateAccount(c *gin.Context)

	// DeleteAccount handles DELETE /admin/ai/accounts/:id.
	DeleteAccount(c *gin.Context)

	// GetAccountStats handles GET /admin/ai/accounts/:id/stats.
	GetAccountStats(c *gin.Context)

	// ResetAccountHealth handles POST /admin/ai/accounts/:id/reset-health.
	ResetAccountHealth(c *gin.Context)
}

// ===== Model Group HTTP Ports =====

// AIModelGroupHttpPort defines model group HTTP handler interface.
type AIModelGroupHttpPort interface {
	// ListGroups handles GET /admin/ai/groups.
	ListGroups(c *gin.Context)

	// GetGroup handles GET /admin/ai/groups/:id.
	GetGroup(c *gin.Context)

	// CreateGroup handles POST /admin/ai/groups.
	CreateGroup(c *gin.Context)

	// UpdateGroup handles PUT /admin/ai/groups/:id.
	UpdateGroup(c *gin.Context)

	// DeleteGroup handles DELETE /admin/ai/groups/:id.
	DeleteGroup(c *gin.Context)
}

// ===== Public API Ports =====

// AIPublicHttpPort defines public AI API handler interface (for listing available models).
type AIPublicHttpPort interface {
	// ListModels handles GET /v1/models (OpenAI compatible).
	ListModels(c *gin.Context)

	// GetModel handles GET /v1/models/:id (OpenAI compatible).
	GetModel(c *gin.Context)
}
