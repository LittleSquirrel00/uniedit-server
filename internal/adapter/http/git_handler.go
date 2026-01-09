package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// GitHandler handles Git HTTP endpoints.
type GitHandler struct {
	domain    inbound.GitDomain
	lfsDomain inbound.GitLFSDomain
	lockDomain inbound.GitLFSLockDomain
	baseURL   string
}

// NewGitHandler creates a new Git handler.
func NewGitHandler(
	domain inbound.GitDomain,
	lfsDomain inbound.GitLFSDomain,
	lockDomain inbound.GitLFSLockDomain,
	baseURL string,
) *GitHandler {
	return &GitHandler{
		domain:    domain,
		lfsDomain: lfsDomain,
		lockDomain: lockDomain,
		baseURL:   baseURL,
	}
}

// RegisterRoutes registers Git routes.
func (h *GitHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	repos := r.Group("/repos")
	{
		repos.Use(authMiddleware)
		repos.POST("", h.CreateRepo)
		repos.GET("", h.ListRepos)
		repos.GET("/:id", h.GetRepo)
		repos.PUT("/:id", h.UpdateRepo)
		repos.DELETE("/:id", h.DeleteRepo)

		// Collaborators
		repos.POST("/:id/collaborators", h.AddCollaborator)
		repos.GET("/:id/collaborators", h.ListCollaborators)
		repos.PUT("/:id/collaborators/:user_id", h.UpdateCollaborator)
		repos.DELETE("/:id/collaborators/:user_id", h.RemoveCollaborator)

		// Pull requests
		repos.POST("/:id/pulls", h.CreatePR)
		repos.GET("/:id/pulls", h.ListPRs)
		repos.GET("/:id/pulls/:number", h.GetPR)
		repos.PUT("/:id/pulls/:number", h.UpdatePR)
		repos.POST("/:id/pulls/:number/merge", h.MergePR)

		// Storage stats
		repos.GET("/:id/storage", h.GetStorageStats)
	}

	// Public repos (no auth required)
	r.GET("/repos/public", h.ListPublicRepos)
}

// ===== Repository Handlers =====

// CreateRepoRequest represents a create repository request.
type CreateRepoRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Type        string `json:"type" binding:"omitempty,oneof=code workflow project"`
	Visibility  string `json:"visibility" binding:"omitempty,oneof=public private"`
	Description string `json:"description" binding:"max=500"`
	LFSEnabled  bool   `json:"lfs_enabled"`
}

// CreateRepo creates a new repository.
func (h *GitHandler) CreateRepo(c *gin.Context) {
	var req CreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	input := &inbound.GitCreateRepoInput{
		Name:        req.Name,
		Type:        model.GitRepoType(req.Type),
		Visibility:  model.GitVisibility(req.Visibility),
		Description: req.Description,
		LFSEnabled:  req.LFSEnabled,
	}

	repo, err := h.domain.CreateRepo(c.Request.Context(), userID, input)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toRepoResponse(repo, h.baseURL))
}

// GetRepo retrieves a repository.
func (h *GitHandler) GetRepo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	repo, err := h.domain.GetRepo(c.Request.Context(), id)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, toRepoResponse(repo, h.baseURL))
}

// ListRepos lists repositories for the current user.
func (h *GitHandler) ListRepos(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	filter := parseRepoFilter(c)
	repos, total, err := h.domain.ListRepos(c.Request.Context(), userID, filter)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repos":       toRepoResponses(repos, h.baseURL),
		"total_count": total,
		"page":        filter.Page,
		"page_size":   filter.PageSize,
	})
}

// ListPublicRepos lists public repositories.
func (h *GitHandler) ListPublicRepos(c *gin.Context) {
	filter := parseRepoFilter(c)
	repos, total, err := h.domain.ListPublicRepos(c.Request.Context(), filter)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repos":       toRepoResponses(repos, h.baseURL),
		"total_count": total,
		"page":        filter.Page,
		"page_size":   filter.PageSize,
	})
}

// UpdateRepoRequest represents an update repository request.
type UpdateRepoRequest struct {
	Name          string `json:"name" binding:"omitempty,min=1,max=100"`
	Description   string `json:"description" binding:"max=500"`
	Visibility    string `json:"visibility" binding:"omitempty,oneof=public private"`
	DefaultBranch string `json:"default_branch" binding:"omitempty,min=1,max=255"`
	LFSEnabled    *bool  `json:"lfs_enabled"`
}

// UpdateRepo updates a repository.
func (h *GitHandler) UpdateRepo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req UpdateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	input := &inbound.GitUpdateRepoInput{
		Name:          req.Name,
		Description:   req.Description,
		Visibility:    model.GitVisibility(req.Visibility),
		DefaultBranch: req.DefaultBranch,
		LFSEnabled:    req.LFSEnabled,
	}

	repo, err := h.domain.UpdateRepo(c.Request.Context(), id, userID, input)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, toRepoResponse(repo, h.baseURL))
}

// DeleteRepo deletes a repository.
func (h *GitHandler) DeleteRepo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.domain.DeleteRepo(c.Request.Context(), id, userID); err != nil {
		handleGitError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ===== Collaborator Handlers =====

// AddCollaboratorRequest represents an add collaborator request.
type AddCollaboratorRequest struct {
	UserID     uuid.UUID `json:"user_id" binding:"required"`
	Permission string    `json:"permission" binding:"required,oneof=read write admin"`
}

// AddCollaborator adds a collaborator to a repository.
func (h *GitHandler) AddCollaborator(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req AddCollaboratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.domain.AddCollaborator(c.Request.Context(), repoID, userID, req.UserID, model.GitPermission(req.Permission)); err != nil {
		handleGitError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// ListCollaborators lists collaborators of a repository.
func (h *GitHandler) ListCollaborators(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	collabs, err := h.domain.ListCollaborators(c.Request.Context(), repoID)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"collaborators": collabs})
}

// UpdateCollaborator updates a collaborator's permission.
func (h *GitHandler) UpdateCollaborator(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req struct {
		Permission string `json:"permission" binding:"required,oneof=read write admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.domain.UpdateCollaborator(c.Request.Context(), repoID, userID, targetUserID, model.GitPermission(req.Permission)); err != nil {
		handleGitError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// RemoveCollaborator removes a collaborator from a repository.
func (h *GitHandler) RemoveCollaborator(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.domain.RemoveCollaborator(c.Request.Context(), repoID, userID, targetUserID); err != nil {
		handleGitError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ===== Pull Request Handlers =====

// CreatePRRequest represents a create pull request request.
type CreatePRRequest struct {
	Title        string `json:"title" binding:"required,min=1,max=255"`
	Description  string `json:"description" binding:"max=10000"`
	SourceBranch string `json:"source_branch" binding:"required,min=1,max=255"`
	TargetBranch string `json:"target_branch" binding:"required,min=1,max=255"`
}

// CreatePR creates a new pull request.
func (h *GitHandler) CreatePR(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req CreatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	input := &inbound.GitCreatePRInput{
		Title:        req.Title,
		Description:  req.Description,
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
	}

	pr, err := h.domain.CreatePR(c.Request.Context(), repoID, userID, input)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pr)
}

// GetPR retrieves a pull request.
func (h *GitHandler) GetPR(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pull request number"})
		return
	}

	pr, err := h.domain.GetPR(c.Request.Context(), repoID, number)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr)
}

// ListPRs lists pull requests for a repository.
func (h *GitHandler) ListPRs(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var status *model.GitPRStatus
	if s := c.Query("status"); s != "" {
		st := model.GitPRStatus(s)
		status = &st
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	prs, total, err := h.domain.ListPRs(c.Request.Context(), repoID, status, limit, offset)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pull_requests": prs,
		"total_count":   total,
	})
}

// UpdatePRRequest represents an update pull request request.
type UpdatePRRequest struct {
	Title       string `json:"title" binding:"omitempty,min=1,max=255"`
	Description string `json:"description" binding:"max=10000"`
	Status      string `json:"status" binding:"omitempty,oneof=open closed"`
}

// UpdatePR updates a pull request.
func (h *GitHandler) UpdatePR(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pull request number"})
		return
	}

	var req UpdatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	input := &inbound.GitUpdatePRInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	}

	pr, err := h.domain.UpdatePR(c.Request.Context(), repoID, number, userID, input)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr)
}

// MergePR merges a pull request.
func (h *GitHandler) MergePR(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pull request number"})
		return
	}

	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	pr, err := h.domain.MergePR(c.Request.Context(), repoID, number, userID)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr)
}

// ===== Storage Handlers =====

// GetStorageStats gets storage statistics for a repository.
func (h *GitHandler) GetStorageStats(c *gin.Context) {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	stats, err := h.domain.GetStorageStats(c.Request.Context(), repoID)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ===== Helper Functions =====

func getUserID(c *gin.Context) uuid.UUID {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

func parseRepoFilter(c *gin.Context) *inbound.GitRepoFilter {
	filter := &inbound.GitRepoFilter{
		Search:   c.Query("search"),
		Page:     1,
		PageSize: 20,
	}

	if t := c.Query("type"); t != "" {
		rt := model.GitRepoType(t)
		filter.Type = &rt
	}

	if v := c.Query("visibility"); v != "" {
		vis := model.GitVisibility(v)
		filter.Visibility = &vis
	}

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		filter.Page = p
	}

	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		filter.PageSize = ps
	}

	return filter
}

func handleGitError(c *gin.Context, err error) {
	switch err.Error() {
	case "repository not found", "pull request not found":
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case "access denied", "not repository owner", "not a collaborator":
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case "repository already exists", "invalid repository name", "source and target branches are the same":
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case "storage quota exceeded":
		c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
	case "pull request is already merged", "pull request is already closed":
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// RepoResponse represents a repository response.
type RepoResponse struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       uuid.UUID  `json:"owner_id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	RepoType      string     `json:"repo_type"`
	Visibility    string     `json:"visibility"`
	Description   string     `json:"description,omitempty"`
	DefaultBranch string     `json:"default_branch"`
	SizeBytes     int64      `json:"size_bytes"`
	LFSEnabled    bool       `json:"lfs_enabled"`
	LFSSizeBytes  int64      `json:"lfs_size_bytes"`
	TotalSize     int64      `json:"total_size"`
	StarsCount    int        `json:"stars_count"`
	ForksCount    int        `json:"forks_count"`
	CloneURL      string     `json:"clone_url,omitempty"`
	LFSURL        string     `json:"lfs_url,omitempty"`
}

func toRepoResponse(repo *model.GitRepo, baseURL string) *RepoResponse {
	resp := &RepoResponse{
		ID:            repo.ID,
		OwnerID:       repo.OwnerID,
		Name:          repo.Name,
		Slug:          repo.Slug,
		RepoType:      string(repo.RepoType),
		Visibility:    string(repo.Visibility),
		Description:   repo.Description,
		DefaultBranch: repo.DefaultBranch,
		SizeBytes:     repo.SizeBytes,
		LFSEnabled:    repo.LFSEnabled,
		LFSSizeBytes:  repo.LFSSizeBytes,
		TotalSize:     repo.TotalSize(),
		StarsCount:    repo.StarsCount,
		ForksCount:    repo.ForksCount,
	}

	if baseURL != "" {
		resp.CloneURL = baseURL + "/git/" + repo.OwnerID.String() + "/" + repo.Slug + ".git"
		if repo.LFSEnabled {
			resp.LFSURL = baseURL + "/lfs/" + repo.OwnerID.String() + "/" + repo.Slug
		}
	}

	return resp
}

func toRepoResponses(repos []*model.GitRepo, baseURL string) []*RepoResponse {
	result := make([]*RepoResponse, len(repos))
	for i, repo := range repos {
		result[i] = toRepoResponse(repo, baseURL)
	}
	return result
}
