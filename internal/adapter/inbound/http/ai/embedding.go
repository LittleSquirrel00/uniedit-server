package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// EmbeddingHandler implements inbound.AIEmbeddingHttpPort.
type EmbeddingHandler struct {
	domain ai.AIDomain
}

// NewEmbeddingHandler creates a new embedding handler.
func NewEmbeddingHandler(domain ai.AIDomain) *EmbeddingHandler {
	return &EmbeddingHandler{domain: domain}
}

// EmbeddingRequest represents an embedding API request.
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingObject represents a single embedding in the response.
type EmbeddingObject struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents token usage for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResponse represents an embedding API response (OpenAI compatible).
type EmbeddingResponse struct {
	Object string             `json:"object"`
	Data   []*EmbeddingObject `json:"data"`
	Model  string             `json:"model"`
	Usage  *EmbeddingUsage    `json:"usage"`
}

// Embed handles embedding requests.
func (h *EmbeddingHandler) Embed(c *gin.Context) {
	var req EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Validate request
	if len(req.Input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input required"})
		return
	}

	// Execute
	resp, err := h.domain.Embed(c.Request.Context(), userID, &model.AIEmbedRequest{
		Model:  req.Model,
		Input:  req.Input,
		UserID: userID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to OpenAI-compatible response format
	data := make([]*EmbeddingObject, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		data[i] = &EmbeddingObject{
			Object:    "embedding",
			Embedding: emb,
			Index:     i,
		}
	}

	var usage *EmbeddingUsage
	if resp.Usage != nil {
		usage = &EmbeddingUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	c.JSON(http.StatusOK, &EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  resp.Model,
		Usage:  usage,
	})
}

// Compile-time interface check
var _ inbound.AIEmbeddingHttpPort = (*EmbeddingHandler)(nil)
