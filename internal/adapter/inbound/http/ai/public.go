package ai

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/port/inbound"
)

// PublicHandler implements inbound.AIPublicHttpPort.
type PublicHandler struct {
	domain ai.AIDomain
}

// NewPublicHandler creates a new public handler.
func NewPublicHandler(domain ai.AIDomain) *PublicHandler {
	return &PublicHandler{domain: domain}
}

// ModelObject represents a model in OpenAI-compatible format.
type ModelObject struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents the list models response (OpenAI compatible).
type ModelsResponse struct {
	Object string         `json:"object"`
	Data   []*ModelObject `json:"data"`
}

// ListModels handles GET /v1/models (OpenAI compatible).
func (h *PublicHandler) ListModels(c *gin.Context) {
	models, err := h.domain.ListEnabledModels(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	data := make([]*ModelObject, len(models))
	for i, m := range models {
		data[i] = &ModelObject{
			ID:      m.ID,
			Object:  "model",
			Created: m.CreatedAt.Unix(),
			OwnedBy: "uniedit",
		}
	}

	c.JSON(http.StatusOK, &ModelsResponse{
		Object: "list",
		Data:   data,
	})
}

// GetModel handles GET /v1/models/:id (OpenAI compatible).
func (h *PublicHandler) GetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model id required"})
		return
	}

	m, err := h.domain.GetModel(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	if !m.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}

	created := m.CreatedAt.Unix()
	if created == 0 {
		created = time.Now().Unix()
	}

	c.JSON(http.StatusOK, &ModelObject{
		ID:      m.ID,
		Object:  "model",
		Created: created,
		OwnedBy: "uniedit",
	})
}

// Compile-time interface check
var _ inbound.AIPublicHttpPort = (*PublicHandler)(nil)
