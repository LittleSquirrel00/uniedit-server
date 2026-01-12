package aiproto

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	aiv1 "github.com/uniedit/server/api/pb/ai"
	commonv1 "github.com/uniedit/server/api/pb/common"
	aihttp "github.com/uniedit/server/internal/adapter/inbound/http/ai"
	"github.com/uniedit/server/internal/transport/protohttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Handler adapts existing AI HTTP handlers to proto-defined interfaces.
type Handler struct {
	chat          *aihttp.ChatHandler
	providerAdmin *aihttp.ProviderAdminHandler
	modelAdmin    *aihttp.ModelAdminHandler
	public        *aihttp.PublicHandler
}

// NewHandler creates a new proto adapter for AI services.
func NewHandler(
	chat *aihttp.ChatHandler,
	providerAdmin *aihttp.ProviderAdminHandler,
	modelAdmin *aihttp.ModelAdminHandler,
	public *aihttp.PublicHandler,
) *Handler {
	return &Handler{
		chat:          chat,
		providerAdmin: providerAdmin,
		modelAdmin:    modelAdmin,
		public:        public,
	}
}

// ===== User AI endpoints =====

func (h *Handler) Chat(c *gin.Context, in *aiv1.ChatRequest) (*aiv1.ChatResponse, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.chat.Chat(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ChatStream(c *gin.Context, in *aiv1.ChatRequest) (*commonv1.Empty, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.chat.ChatStream(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListModels(c *gin.Context, _ *commonv1.Empty) (*aiv1.ListModelsResponse, error) {
	h.public.ListModels(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetModel(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.Model, error) {
	// Path vars already bound; delegate directly.
	h.public.GetModel(c)
	return nil, protohttp.ErrHandled
}

// ===== Admin AI endpoints =====

func (h *Handler) ListProviders(c *gin.Context, _ *commonv1.Empty) (*aiv1.ListProvidersResponse, error) {
	h.providerAdmin.ListProviders(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) CreateProvider(c *gin.Context, in *aiv1.CreateProviderRequest) (*aiv1.Provider, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.providerAdmin.CreateProvider(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetProvider(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.Provider, error) {
	h.providerAdmin.GetProvider(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdateProvider(c *gin.Context, in *aiv1.UpdateProviderRequest) (*aiv1.Provider, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.providerAdmin.UpdateProvider(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) DeleteProvider(c *gin.Context, in *aiv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	h.providerAdmin.DeleteProvider(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) SyncModels(c *gin.Context, in *aiv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	h.providerAdmin.SyncModels(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) HealthCheck(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.HealthCheckResponse, error) {
	h.providerAdmin.HealthCheck(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListAllModels(c *gin.Context, _ *commonv1.Empty) (*aiv1.ListAllModelsResponse, error) {
	h.modelAdmin.ListModels(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) CreateModel(c *gin.Context, in *aiv1.CreateModelRequest) (*aiv1.Model, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.modelAdmin.CreateModel(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetAdminModel(c *gin.Context, in *aiv1.GetByIDRequest) (*aiv1.Model, error) {
	h.modelAdmin.GetModel(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdateModel(c *gin.Context, in *aiv1.UpdateModelRequest) (*aiv1.Model, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.modelAdmin.UpdateModel(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) DeleteModel(c *gin.Context, in *aiv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	h.modelAdmin.DeleteModel(c)
	return nil, protohttp.ErrHandled
}

// resetBody rehydrates gin.Context request body from the proto message for downstream JSON binding.
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
