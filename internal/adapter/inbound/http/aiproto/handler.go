package aiproto

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	aiv1 "github.com/uniedit/server/api/pb/ai"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	aiDomain ai.AIDomain
}

func NewHandler(aiDomain ai.AIDomain) *Handler {
	return &Handler{aiDomain: aiDomain}
}

// ===== AIService =====

func (h *Handler) Chat(c *gin.Context, in *aiv1.ChatRequest) (*aiv1.ChatResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	resp, err := h.aiDomain.Chat(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) ChatStream(c *gin.Context, in *aiv1.ChatRequest) (*commonv1.Empty, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	if in != nil {
		in.Stream = true
	}

	chunks, routingInfo, err := h.aiDomain.ChatStream(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapAIError(err)
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	if routingInfo != nil {
		data, _ := json.Marshal(routingInfo)
		_, _ = fmt.Fprintf(c.Writer, "event: routing\ndata: %s\n\n", data)
		c.Writer.Flush()
	}

	for chunk := range chunks {
		data, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()

		select {
		case <-c.Request.Context().Done():
			return nil, protohttp.ErrHandled
		default:
		}
	}

	_, _ = fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListModels(c *gin.Context, _ *commonv1.Empty) (*aiv1.ListModelsResponse, error) {
	resp, err := h.aiDomain.ListEnabledModels(c.Request.Context())
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) GetModel(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.Model, error) {
	if in.GetId() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "model id is required"}
	}

	m, err := h.aiDomain.GetModel(c.Request.Context(), in.GetId())
	if err != nil {
		return nil, mapAIError(err)
	}
	return m, nil
}

// ===== AIAdminService =====

func (h *Handler) ListProviders(c *gin.Context, _ *commonv1.Empty) (*aiv1.ListProvidersResponse, error) {
	resp, err := h.aiDomain.ListProviders(c.Request.Context())
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) CreateProvider(c *gin.Context, in *aiv1.CreateProviderRequest) (*aiv1.Provider, error) {
	p, err := h.aiDomain.CreateProvider(c.Request.Context(), in)
	if err != nil {
		return nil, mapAIError(err)
	}
	c.Status(http.StatusCreated)
	return p, nil
}

func (h *Handler) GetProvider(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.Provider, error) {
	id, err := parseUUID(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid provider id", Err: err}
	}

	p, err := h.aiDomain.GetProvider(c.Request.Context(), id)
	if err != nil {
		return nil, mapAIError(err)
	}
	return p, nil
}

func (h *Handler) UpdateProvider(c *gin.Context, in *aiv1.UpdateProviderRequest) (*aiv1.Provider, error) {
	id, err := parseUUID(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid provider id", Err: err}
	}

	p, err := h.aiDomain.UpdateProvider(c.Request.Context(), id, in)
	if err != nil {
		return nil, mapAIError(err)
	}
	return p, nil
}

func (h *Handler) DeleteProvider(c *gin.Context, in *aiv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	id, err := parseUUID(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid provider id", Err: err}
	}

	resp, err := h.aiDomain.DeleteProvider(c.Request.Context(), id)
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) SyncModels(c *gin.Context, in *aiv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	id, err := parseUUID(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid provider id", Err: err}
	}

	resp, err := h.aiDomain.SyncModels(c.Request.Context(), id)
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) HealthCheck(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.HealthCheckResponse, error) {
	id, err := parseUUID(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid provider id", Err: err}
	}

	resp, err := h.aiDomain.ProviderHealthCheck(c.Request.Context(), id)
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) ListAllModels(c *gin.Context, _ *commonv1.Empty) (*aiv1.ListAllModelsResponse, error) {
	resp, err := h.aiDomain.ListModels(c.Request.Context())
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func (h *Handler) CreateModel(c *gin.Context, in *aiv1.CreateModelRequest) (*aiv1.Model, error) {
	m, err := h.aiDomain.CreateModel(c.Request.Context(), in)
	if err != nil {
		return nil, mapAIError(err)
	}
	c.Status(http.StatusCreated)
	return m, nil
}

func (h *Handler) GetAdminModel(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.Model, error) {
	if in.GetId() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "model id is required"}
	}

	m, err := h.aiDomain.GetAdminModel(c.Request.Context(), in.GetId())
	if err != nil {
		return nil, mapAIError(err)
	}
	return m, nil
}

func (h *Handler) UpdateModel(c *gin.Context, in *aiv1.UpdateModelRequest) (*aiv1.Model, error) {
	if in.GetId() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "model id is required"}
	}

	m, err := h.aiDomain.UpdateModel(c.Request.Context(), in.GetId(), in)
	if err != nil {
		return nil, mapAIError(err)
	}
	return m, nil
}

func (h *Handler) DeleteModel(c *gin.Context, in *aiv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	if in.GetId() == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "model id is required"}
	}
	resp, err := h.aiDomain.DeleteModel(c.Request.Context(), in.GetId())
	if err != nil {
		return nil, mapAIError(err)
	}
	return resp, nil
}

func requireUserID(c *gin.Context) (uuid.UUID, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return uuid.Nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}
	return userID, nil
}

func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, fmt.Errorf("empty uuid")
	}
	return uuid.Parse(s)
}

func mapAIError(err error) error {
	switch {
	case errors.Is(err, ai.ErrProviderNotFound),
		errors.Is(err, ai.ErrModelNotFound),
		errors.Is(err, ai.ErrAccountNotFound),
		errors.Is(err, ai.ErrGroupNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrProviderDisabled),
		errors.Is(err, ai.ErrModelDisabled),
		errors.Is(err, ai.ErrAccountDisabled),
		errors.Is(err, ai.ErrGroupDisabled):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "disabled", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrModelNotSupported),
		errors.Is(err, ai.ErrAdapterNotSupported):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "not_supported", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrEmptyMessages),
		errors.Is(err, ai.ErrEmptyInput),
		errors.Is(err, ai.ErrInvalidRequest):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrRateLimitExceeded):
		return &protohttp.HTTPError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrQuotaExceeded):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "quota_exceeded", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrInsufficientCredits):
		return &protohttp.HTTPError{Status: http.StatusPaymentRequired, Code: "insufficient_credits", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrNoAvailableModels),
		errors.Is(err, ai.ErrNoAvailableAccount),
		errors.Is(err, ai.ErrAdapterNotFound),
		errors.Is(err, ai.ErrProviderUnhealthy),
		errors.Is(err, ai.ErrAccountUnhealthy),
		errors.Is(err, ai.ErrRoutingFailed),
		errors.Is(err, ai.ErrAllFallbacksFailed):
		return &protohttp.HTTPError{Status: http.StatusServiceUnavailable, Code: "unavailable", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrUpstreamError):
		return &protohttp.HTTPError{Status: http.StatusBadGateway, Code: "upstream_error", Message: err.Error(), Err: err}
	case errors.Is(err, ai.ErrTimeout):
		return &protohttp.HTTPError{Status: http.StatusGatewayTimeout, Code: "timeout", Message: err.Error(), Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Err: err}
	}
}
