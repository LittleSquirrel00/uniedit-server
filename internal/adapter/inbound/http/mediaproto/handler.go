package mediaproto

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
	"github.com/uniedit/server/internal/domain/media"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	media inbound.MediaDomain
}

func NewHandler(media inbound.MediaDomain) *Handler {
	return &Handler{media: media}
}

// ===== MediaService =====

func (h *Handler) GenerateImage(c *gin.Context, in *mediav1.GenerateImageRequest) (*mediav1.GenerateImageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.media.GenerateImage(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapMediaError(err)
	}

	return out, nil
}

func (h *Handler) GenerateVideo(c *gin.Context, in *mediav1.GenerateVideoRequest) (*mediav1.VideoGenerationStatus, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.media.GenerateVideo(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapMediaError(err)
	}

	return out, nil
}

func (h *Handler) GetVideoStatus(c *gin.Context, in *mediav1.GetByTaskIDRequest) (*mediav1.VideoGenerationStatus, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.media.GetVideoStatus(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapMediaError(err)
	}
	return out, nil
}

func (h *Handler) ListTasks(c *gin.Context, in *mediav1.ListTasksRequest) (*mediav1.ListTasksResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.media.ListTasks(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapMediaError(err)
	}
	return out, nil
}

func (h *Handler) GetTask(c *gin.Context, in *mediav1.GetByTaskIDRequest) (*mediav1.MediaTask, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	task, err := h.media.GetTask(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapMediaError(err)
	}
	return task, nil
}

func (h *Handler) CancelTask(c *gin.Context, in *mediav1.GetByTaskIDRequest) (*commonv1.Empty, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.media.CancelTask(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapMediaError(err)
	}

	c.Status(http.StatusNoContent)
	return out, nil
}

// ===== MediaAdminService =====

func (h *Handler) ListProviders(c *gin.Context, _ *commonv1.Empty) (*mediav1.ListProvidersResponse, error) {
	out, err := h.media.ListProviders(c.Request.Context(), &commonv1.Empty{})
	if err != nil {
		return nil, mapMediaError(err)
	}
	return out, nil
}

func (h *Handler) GetProvider(c *gin.Context, in *mediav1.GetByIDRequest) (*mediav1.MediaProvider, error) {
	p, err := h.media.GetProvider(c.Request.Context(), in)
	if err != nil {
		return nil, mapMediaError(err)
	}
	return p, nil
}

func (h *Handler) ListModels(c *gin.Context, in *mediav1.ListModelsRequest) (*mediav1.ListModelsResponse, error) {
	out, err := h.media.ListModels(c.Request.Context(), in)
	if err != nil {
		return nil, mapMediaError(err)
	}
	return out, nil
}

func requireUserID(c *gin.Context) (uuid.UUID, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return uuid.Nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}
	return userID, nil
}

func mapMediaError(err error) error {
	switch {
	case errors.Is(err, media.ErrProviderNotFound),
		errors.Is(err, media.ErrModelNotFound),
		errors.Is(err, media.ErrTaskNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: err.Error(), Err: err}
	case errors.Is(err, media.ErrTaskNotOwned):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: err.Error(), Err: err}
	case errors.Is(err, media.ErrInvalidInput),
		errors.Is(err, media.ErrCapabilityNotSupported):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: err.Error(), Err: err}
	case errors.Is(err, media.ErrNoAdapterFound),
		errors.Is(err, media.ErrProviderUnhealthy),
		errors.Is(err, media.ErrNoHealthyProvider):
		return &protohttp.HTTPError{Status: http.StatusServiceUnavailable, Code: "unavailable", Message: err.Error(), Err: err}
	case errors.Is(err, media.ErrTaskAlreadyCompleted),
		errors.Is(err, media.ErrTaskAlreadyCancelled):
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "conflict", Message: err.Error(), Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: fmt.Sprintf("Internal server error"), Err: err}
	}
}
