package gitproto

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonv1 "github.com/uniedit/server/api/pb/common"
	gitv1 "github.com/uniedit/server/api/pb/git"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	git inbound.GitDomain
}

func NewHandler(git inbound.GitDomain) *Handler {
	return &Handler{git: git}
}

// ===== Repo operations =====

func (h *Handler) CreateRepo(c *gin.Context, in *gitv1.CreateRepoRequest) (*gitv1.Repo, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	repo, err := h.git.CreateRepo(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}

	c.Status(http.StatusCreated)
	return repo, nil
}

func (h *Handler) ListRepos(c *gin.Context, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	resp, err := h.git.ListRepos(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return resp, nil
}

func (h *Handler) GetRepo(c *gin.Context, in *gitv1.GetByIDRequest) (*gitv1.Repo, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	repo, err := h.git.GetRepo(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return repo, nil
}

func (h *Handler) UpdateRepo(c *gin.Context, in *gitv1.UpdateRepoRequest) (*gitv1.Repo, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	repo, err := h.git.UpdateRepo(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return repo, nil
}

func (h *Handler) DeleteRepo(c *gin.Context, in *gitv1.GetByIDRequest) (*commonv1.Empty, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.git.DeleteRepo(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}

	c.Status(http.StatusNoContent)
	return out, nil
}

func (h *Handler) GetStorageStats(c *gin.Context, in *gitv1.GetByIDRequest) (*gitv1.StorageStats, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	stats, err := h.git.GetStorageStats(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return stats, nil
}

// ===== Collaborators =====

func (h *Handler) AddCollaborator(c *gin.Context, in *gitv1.AddCollaboratorRequest) (*commonv1.Empty, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.git.AddCollaborator(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}

	c.Status(http.StatusNoContent)
	return out, nil
}

func (h *Handler) ListCollaborators(c *gin.Context, in *gitv1.GetByIDRequest) (*gitv1.ListCollaboratorsResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.git.ListCollaborators(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return out, nil
}

func (h *Handler) UpdateCollaborator(c *gin.Context, in *gitv1.UpdateCollaboratorRequest) (*commonv1.Empty, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.git.UpdateCollaborator(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}

	c.Status(http.StatusNoContent)
	return out, nil
}

func (h *Handler) RemoveCollaborator(c *gin.Context, in *gitv1.RemoveCollaboratorRequest) (*commonv1.Empty, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.git.RemoveCollaborator(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}

	c.Status(http.StatusNoContent)
	return out, nil
}

// ===== Pull Requests =====

func (h *Handler) CreatePR(c *gin.Context, in *gitv1.CreatePRRequest) (*gitv1.PullRequest, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	pr, err := h.git.CreatePR(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}

	c.Status(http.StatusCreated)
	return pr, nil
}

func (h *Handler) ListPRs(c *gin.Context, in *gitv1.ListPRsRequest) (*gitv1.ListPRsResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.git.ListPRs(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return out, nil
}

func (h *Handler) GetPR(c *gin.Context, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	pr, err := h.git.GetPR(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return pr, nil
}

func (h *Handler) UpdatePR(c *gin.Context, in *gitv1.UpdatePRRequest) (*gitv1.PullRequest, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	pr, err := h.git.UpdatePR(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return pr, nil
}

func (h *Handler) MergePR(c *gin.Context, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	pr, err := h.git.MergePR(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapGitError(err)
	}
	return pr, nil
}

// ===== Public =====

func (h *Handler) ListPublicRepos(c *gin.Context, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error) {
	out, err := h.git.ListPublicRepos(c.Request.Context(), in)
	if err != nil {
		return nil, mapGitError(err)
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

func mapGitError(err error) error {
	switch err.Error() {
	case "repository not found", "pull request not found":
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: err.Error(), Err: err}
	case "access denied", "not repository owner", "not a collaborator":
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: err.Error(), Err: err}
	case "invalid request", "invalid permission level", "repository already exists", "invalid repository name", "source and target branches are the same":
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: err.Error(), Err: err}
	case "storage quota exceeded":
		return &protohttp.HTTPError{Status: http.StatusPaymentRequired, Code: "quota_exceeded", Message: err.Error(), Err: err}
	case "pull request is already merged", "pull request is already closed":
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "conflict", Message: err.Error(), Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Err: err}
	}
}
