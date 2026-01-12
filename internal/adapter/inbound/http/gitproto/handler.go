package gitproto

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	commonv1 "github.com/uniedit/server/api/pb/common"
	gitv1 "github.com/uniedit/server/api/pb/git"
	githttp "github.com/uniedit/server/internal/adapter/inbound/http/git"
	"github.com/uniedit/server/internal/transport/protohttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Handler adapts Git HTTP handlers to proto-defined interfaces.
type Handler struct {
	git *githttp.Handler
}

// NewHandler creates a new Git proto adapter.
func NewHandler(git *githttp.Handler) *Handler {
	return &Handler{git: git}
}

// ===== Repo operations =====

func (h *Handler) CreateRepo(c *gin.Context, in *gitv1.CreateRepoRequest) (*gitv1.Repo, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.git.CreateRepo(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListRepos(c *gin.Context, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error) {
	h.git.ListRepos(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetRepo(c *gin.Context, in *gitv1.GetByIDRequest) (*gitv1.Repo, error) {
	h.git.GetRepo(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdateRepo(c *gin.Context, in *gitv1.UpdateRepoRequest) (*gitv1.Repo, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.git.UpdateRepo(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) DeleteRepo(c *gin.Context, in *gitv1.GetByIDRequest) (*commonv1.Empty, error) {
	h.git.DeleteRepo(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetStorageStats(c *gin.Context, in *gitv1.GetByIDRequest) (*gitv1.StorageStats, error) {
	h.git.GetStorageStats(c)
	return nil, protohttp.ErrHandled
}

// ===== Collaborators =====

func (h *Handler) AddCollaborator(c *gin.Context, in *gitv1.AddCollaboratorRequest) (*commonv1.Empty, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.git.AddCollaborator(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListCollaborators(c *gin.Context, in *gitv1.GetByIDRequest) (*gitv1.ListCollaboratorsResponse, error) {
	h.git.ListCollaborators(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdateCollaborator(c *gin.Context, in *gitv1.UpdateCollaboratorRequest) (*commonv1.Empty, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.git.UpdateCollaborator(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) RemoveCollaborator(c *gin.Context, in *gitv1.RemoveCollaboratorRequest) (*commonv1.Empty, error) {
	h.git.RemoveCollaborator(c)
	return nil, protohttp.ErrHandled
}

// ===== Pull Requests =====

func (h *Handler) CreatePR(c *gin.Context, in *gitv1.CreatePRRequest) (*gitv1.PullRequest, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.git.CreatePR(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListPRs(c *gin.Context, in *gitv1.ListPRsRequest) (*gitv1.ListPRsResponse, error) {
	h.git.ListPRs(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetPR(c *gin.Context, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error) {
	h.git.GetPR(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdatePR(c *gin.Context, in *gitv1.UpdatePRRequest) (*gitv1.PullRequest, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.git.UpdatePR(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) MergePR(c *gin.Context, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error) {
	h.git.MergePR(c)
	return nil, protohttp.ErrHandled
}

// ===== Public =====

func (h *Handler) ListPublicRepos(c *gin.Context, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error) {
	h.git.ListPublicRepos(c)
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

