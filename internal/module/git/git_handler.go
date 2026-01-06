package git

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/module/git/protocol"
	"github.com/uniedit/server/internal/module/git/storage"
)

// GitAuthenticator defines the interface for Git client authentication.
type GitAuthenticator interface {
	// AuthenticateBasic authenticates using HTTP Basic Auth (username:token)
	AuthenticateBasic(ctx *gin.Context, username, password string) (*uuid.UUID, error)
}

// GitHandler handles Git HTTP protocol requests.
type GitHandler struct {
	service        ServiceInterface
	r2Client       *storage.R2Client
	authenticator  GitAuthenticator
	smartHTTP      *protocol.SmartHTTPHandler
	logger         *zap.Logger
}

// NewGitHandler creates a new Git HTTP handler.
func NewGitHandler(
	service ServiceInterface,
	r2Client *storage.R2Client,
	authenticator GitAuthenticator,
	logger *zap.Logger,
) *GitHandler {
	return &GitHandler{
		service:       service,
		r2Client:      r2Client,
		authenticator: authenticator,
		smartHTTP:     protocol.NewSmartHTTPHandler(logger),
		logger:        logger,
	}
}

// RegisterGitRoutes registers Git HTTP protocol routes.
func (h *GitHandler) RegisterGitRoutes(r *gin.RouterGroup) {
	// Git Smart HTTP endpoints
	// Pattern: /git/:owner/:repo.git/...
	git := r.Group("/git")
	{
		git.GET("/:owner/:repo/info/refs", h.InfoRefs)
		git.POST("/:owner/:repo/git-upload-pack", h.UploadPack)
		git.POST("/:owner/:repo/git-receive-pack", h.ReceivePack)
	}
}

// InfoRefs handles Git info/refs discovery.
func (h *GitHandler) InfoRefs(c *gin.Context) {
	reqCtx, err := h.resolveGitRequest(c)
	if err != nil {
		return // Error already handled
	}

	service := c.Query("service")
	svc, err := protocol.ParseService(service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_service"})
		return
	}

	data, contentType, err := h.smartHTTP.InfoRefs(c.Request.Context(), reqCtx, svc)
	if err != nil {
		h.logger.Error("info/refs failed", zap.Error(err))
		h.sendGitAuthError(c)
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, contentType, data)
}

// UploadPack handles git-upload-pack (fetch/clone).
func (h *GitHandler) UploadPack(c *gin.Context) {
	reqCtx, err := h.resolveGitRequest(c)
	if err != nil {
		return
	}

	contentType := protocol.GetContentType(protocol.ServiceUploadPack, false)
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-cache")

	if err := h.smartHTTP.UploadPack(c.Request.Context(), reqCtx, c.Request.Body, c.Writer); err != nil {
		h.logger.Error("upload-pack failed", zap.Error(err))
		// Response may already be partially written
	}
}

// ReceivePack handles git-receive-pack (push).
func (h *GitHandler) ReceivePack(c *gin.Context) {
	reqCtx, err := h.resolveGitRequest(c)
	if err != nil {
		return
	}

	if !reqCtx.CanWrite {
		h.sendGitAuthError(c)
		return
	}

	contentType := protocol.GetContentType(protocol.ServiceReceivePack, false)
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-cache")

	onComplete := func(pushedAt time.Time) {
		// Update pushed_at timestamp
		if err := h.service.UpdatePushedAt(c.Request.Context(), reqCtx.RepoID); err != nil {
			h.logger.Error("failed to update pushed_at", zap.Error(err))
		}
	}

	if err := h.smartHTTP.ReceivePack(c.Request.Context(), reqCtx, c.Request.Body, c.Writer, onComplete); err != nil {
		h.logger.Error("receive-pack failed", zap.Error(err))
	}
}

// resolveGitRequest resolves the repository and authenticates the user.
func (h *GitHandler) resolveGitRequest(c *gin.Context) (*protocol.RequestContext, error) {
	ownerParam := c.Param("owner")
	repoSlug := c.Param("repo")

	// Remove .git suffix if present
	repoSlug = strings.TrimSuffix(repoSlug, ".git")

	// Parse owner ID
	ownerID, err := uuid.Parse(ownerParam)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository_not_found"})
		return nil, err
	}

	// Get repository
	repo, err := h.service.GetRepoByOwnerAndSlug(c.Request.Context(), ownerID, repoSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository_not_found"})
		return nil, err
	}

	// Authenticate user (optional for public repos read access)
	var userID *uuid.UUID
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		userID = h.authenticateRequest(c, authHeader)
	}

	// Check access
	canRead, _ := h.service.CanAccess(c.Request.Context(), repo.ID, userID, PermissionRead)
	canWrite := false
	if userID != nil {
		canWrite, _ = h.service.CanAccess(c.Request.Context(), repo.ID, userID, PermissionWrite)
	}

	// For private repos, require authentication
	if repo.Visibility == VisibilityPrivate && !canRead {
		h.sendGitAuthError(c)
		return nil, ErrAccessDenied
	}

	// Get R2 filesystem for repo
	fs := storage.NewR2Filesystem(h.r2Client, repo.StoragePath)

	return &protocol.RequestContext{
		RepoID:     repo.ID,
		UserID:     userID,
		CanRead:    canRead,
		CanWrite:   canWrite,
		Filesystem: fs,
	}, nil
}

// authenticateRequest extracts credentials from Authorization header.
func (h *GitHandler) authenticateRequest(c *gin.Context, authHeader string) *uuid.UUID {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return nil
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil
	}

	username, password := parts[0], parts[1]

	if h.authenticator != nil {
		userID, err := h.authenticator.AuthenticateBasic(c, username, password)
		if err != nil {
			h.logger.Debug("git auth failed", zap.String("username", username), zap.Error(err))
			return nil
		}
		return userID
	}

	// Fallback: username is user ID, password is API key/token
	// This is a simplified auth - in production, integrate with auth module
	userID, err := uuid.Parse(username)
	if err != nil {
		return nil
	}
	_ = password // Would validate against auth service
	return &userID
}

// sendGitAuthError sends a 401 response with WWW-Authenticate header.
func (h *GitHandler) sendGitAuthError(c *gin.Context) {
	c.Header("WWW-Authenticate", `Basic realm="Git"`)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication_required"})
}
