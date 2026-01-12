package mediaproto

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	commonv1 "github.com/uniedit/server/api/pb/common"
	mediav1 "github.com/uniedit/server/api/pb/media"
	mediahttp "github.com/uniedit/server/internal/adapter/inbound/http/media"
	"github.com/uniedit/server/internal/transport/protohttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Handler adapts media HTTP handlers to proto-defined interfaces.
type Handler struct {
	media *mediahttp.Handler
}

// NewHandler creates a new media proto adapter.
func NewHandler(media *mediahttp.Handler) *Handler {
	return &Handler{media: media}
}

// ===== MediaService =====

func (h *Handler) GenerateImage(c *gin.Context, in *mediav1.GenerateImageRequest) (*mediav1.GenerateImageResponse, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.media.GenerateImage(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GenerateVideo(c *gin.Context, in *mediav1.GenerateVideoRequest) (*mediav1.VideoGenerationStatus, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.media.GenerateVideo(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetVideoStatus(c *gin.Context, in *mediav1.GetByTaskIDRequest) (*mediav1.VideoGenerationStatus, error) {
	h.media.GetVideoStatus(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListTasks(c *gin.Context, in *mediav1.ListTasksRequest) (*mediav1.ListTasksResponse, error) {
	h.media.ListTasks(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetTask(c *gin.Context, in *mediav1.GetByTaskIDRequest) (*mediav1.MediaTask, error) {
	h.media.GetTask(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) CancelTask(c *gin.Context, in *mediav1.GetByTaskIDRequest) (*commonv1.Empty, error) {
	h.media.CancelTask(c)
	return nil, protohttp.ErrHandled
}

// ===== MediaAdminService =====

func (h *Handler) ListProviders(c *gin.Context, in *commonv1.Empty) (*mediav1.ListProvidersResponse, error) {
	h.media.ListProviders(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetProvider(c *gin.Context, in *mediav1.GetByIDRequest) (*mediav1.MediaProvider, error) {
	h.media.GetProvider(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListModels(c *gin.Context, in *mediav1.ListModelsRequest) (*mediav1.ListModelsResponse, error) {
	h.media.ListModels(c)
	return nil, protohttp.ErrHandled
}

func resetBody(c *gin.Context, msg proto.Message) error {
	if c == nil || c.Request == nil || msg == nil {
		return nil
	}

	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return err
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	c.Request.ContentLength = int64(len(data))
	if c.Request.Header.Get("Content-Type") == "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return nil
}

