package git

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for git repositories.
type Handler struct {
	service ServiceInterface
	baseURL string // For generating clone URLs
}

// NewHandler creates a new git handler.
func NewHandler(service ServiceInterface, baseURL string) *Handler {
	return &Handler{
		service: service,
		baseURL: baseURL,
	}
}

// RegisterRoutes registers public git routes (no auth required).
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	repos := r.Group("/repos")
	{
		// Public endpoints - no auth required
		repos.GET("/public", h.ListPublicRepos)
		repos.GET("/:owner/:repo", h.GetRepo)

		// Pull Requests - read access (uses getUserIDOptional)
		repos.GET("/:owner/:repo/pulls", h.ListPRs)
		repos.GET("/:owner/:repo/pulls/:number", h.GetPR)
	}
}

// RegisterProtectedRoutes registers git routes that require authentication.
func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	repos := r.Group("/repos")
	{
		// Repository CRUD - requires auth
		repos.POST("", h.CreateRepo)
		repos.GET("", h.ListRepos)
		repos.PATCH("/:owner/:repo", h.UpdateRepo)
		repos.DELETE("/:owner/:repo", h.DeleteRepo)

		// Collaborators - requires auth
		repos.GET("/:owner/:repo/collaborators", h.ListCollaborators)
		repos.PUT("/:owner/:repo/collaborators/:user_id", h.AddCollaborator)
		repos.PATCH("/:owner/:repo/collaborators/:user_id", h.UpdateCollaborator)
		repos.DELETE("/:owner/:repo/collaborators/:user_id", h.RemoveCollaborator)

		// Pull Requests - write operations require auth
		repos.POST("/:owner/:repo/pulls", h.CreatePR)
		repos.PATCH("/:owner/:repo/pulls/:number", h.UpdatePR)
		repos.POST("/:owner/:repo/pulls/:number/merge", h.MergePR)

		// Storage stats - requires auth
		repos.GET("/:owner/:repo/storage", h.GetStorageStats)
	}

	// User storage stats - requires auth
	r.GET("/storage", h.GetUserStorageStats)
}

// --- Repository Handlers ---

// CreateRepo creates a new repository.
//
//	@Summary		Create repository
//	@Description	Create a new git repository
//	@Tags			Git
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateRepoRequest	true	"Create repository request"
//	@Success		201		{object}	RepoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/repos [post]
func (h *Handler) CreateRepo(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo, err := h.service.CreateRepo(c.Request.Context(), userID, &req)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusCreated, repo.ToResponse(h.baseURL))
}

// GetRepo retrieves a repository.
//
//	@Summary		Get repository
//	@Description	Get details of a specific repository
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string	true	"Owner ID"
//	@Param			repo	path		string	true	"Repository slug"
//	@Success		200		{object}	RepoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo} [get]
func (h *Handler) GetRepo(c *gin.Context) {
	ownerID, repo, err := h.resolveRepo(c)
	if err != nil {
		return // Error already handled
	}

	// Check read access
	userID := getUserIDOptional(c)
	canAccess, err := h.service.CanAccess(c.Request.Context(), repo.ID, userID, PermissionRead)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	if !canAccess {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository_not_found"})
		return
	}

	_ = ownerID // Used for resolution
	c.JSON(http.StatusOK, repo.ToResponse(h.baseURL))
}

// ListRepos lists repositories for the authenticated user.
//
//	@Summary		List repositories
//	@Description	Get all repositories for the current user
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			page_size	query		int		false	"Page size"			default(20)
//	@Param			search		query		string	false	"Search query"
//	@Param			type		query		string	false	"Repository type"
//	@Param			visibility	query		string	false	"Visibility filter"
//	@Success		200			{object}	ListReposResponse
//	@Failure		401			{object}	map[string]string
//	@Router			/repos [get]
func (h *Handler) ListRepos(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	filter := parseRepoFilter(c)
	repos, total, err := h.service.ListRepos(c.Request.Context(), userID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list repos"})
		return
	}

	responses := make([]*RepoResponse, len(repos))
	for i, repo := range repos {
		responses[i] = repo.ToResponse(h.baseURL)
	}

	c.JSON(http.StatusOK, ListReposResponse{
		Repos:      responses,
		TotalCount: total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	})
}

// ListPublicRepos lists public repositories.
//
//	@Summary		List public repositories
//	@Description	Get all public repositories
//	@Tags			Git
//	@Produce		json
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			page_size	query		int		false	"Page size"		default(20)
//	@Param			search		query		string	false	"Search query"
//	@Success		200			{object}	ListReposResponse
//	@Router			/repos/public [get]
func (h *Handler) ListPublicRepos(c *gin.Context) {
	filter := parseRepoFilter(c)
	repos, total, err := h.service.ListPublicRepos(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list repos"})
		return
	}

	responses := make([]*RepoResponse, len(repos))
	for i, repo := range repos {
		responses[i] = repo.ToResponse(h.baseURL)
	}

	c.JSON(http.StatusOK, ListReposResponse{
		Repos:      responses,
		TotalCount: total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	})
}

// UpdateRepo updates a repository.
//
//	@Summary		Update repository
//	@Description	Update repository settings
//	@Tags			Git
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string				true	"Owner ID"
//	@Param			repo	path		string				true	"Repository slug"
//	@Param			request	body		UpdateRepoRequest	true	"Update request"
//	@Success		200		{object}	RepoResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo} [patch]
func (h *Handler) UpdateRepo(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	var req UpdateRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.service.UpdateRepo(c.Request.Context(), repo.ID, userID, &req)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated.ToResponse(h.baseURL))
}

// DeleteRepo deletes a repository.
//
//	@Summary		Delete repository
//	@Description	Permanently delete a repository
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path	string	true	"Owner ID"
//	@Param			repo	path	string	true	"Repository slug"
//	@Success		204		"No Content"
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo} [delete]
func (h *Handler) DeleteRepo(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	if err := h.service.DeleteRepo(c.Request.Context(), repo.ID, userID); err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// --- Collaborator Handlers ---

// ListCollaborators lists collaborators.
//
//	@Summary		List collaborators
//	@Description	Get all collaborators for a repository
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string	true	"Owner ID"
//	@Param			repo	path		string	true	"Repository slug"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/collaborators [get]
func (h *Handler) ListCollaborators(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	// Must be owner or collaborator to list
	if err := h.service.CheckAccess(c.Request.Context(), repo.ID, userID, PermissionRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access_denied"})
		return
	}

	collabs, err := h.service.ListCollaborators(c.Request.Context(), repo.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list collaborators"})
		return
	}

	responses := make([]*CollaboratorResponse, len(collabs))
	for i, collab := range collabs {
		responses[i] = collab.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"collaborators": responses})
}

// AddCollaborator adds a collaborator.
//
//	@Summary		Add collaborator
//	@Description	Add a collaborator to a repository
//	@Tags			Git
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string					true	"Owner ID"
//	@Param			repo	path		string					true	"Repository slug"
//	@Param			user_id	path		string					true	"User ID to add"
//	@Param			request	body		AddCollaboratorRequest	true	"Collaborator request"
//	@Success		201		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/collaborators/{user_id} [put]
func (h *Handler) AddCollaborator(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req AddCollaboratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddCollaborator(c.Request.Context(), repo.ID, userID, targetUserID, Permission(req.Permission)); err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "collaborator added"})
}

// UpdateCollaborator updates a collaborator's permission.
//
//	@Summary		Update collaborator
//	@Description	Update a collaborator's permission level
//	@Tags			Git
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string					true	"Owner ID"
//	@Param			repo	path		string					true	"Repository slug"
//	@Param			user_id	path		string					true	"User ID"
//	@Param			request	body		AddCollaboratorRequest	true	"Permission update"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/collaborators/{user_id} [patch]
func (h *Handler) UpdateCollaborator(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req AddCollaboratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateCollaborator(c.Request.Context(), repo.ID, userID, targetUserID, Permission(req.Permission)); err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "collaborator updated"})
}

// RemoveCollaborator removes a collaborator.
//
//	@Summary		Remove collaborator
//	@Description	Remove a collaborator from a repository
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path	string	true	"Owner ID"
//	@Param			repo	path	string	true	"Repository slug"
//	@Param			user_id	path	string	true	"User ID to remove"
//	@Success		204		"No Content"
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/collaborators/{user_id} [delete]
func (h *Handler) RemoveCollaborator(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.service.RemoveCollaborator(c.Request.Context(), repo.ID, userID, targetUserID); err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// --- Pull Request Handlers ---

// CreatePR creates a pull request.
//
//	@Summary		Create pull request
//	@Description	Create a new pull request
//	@Tags			Git
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string			true	"Owner ID"
//	@Param			repo	path		string			true	"Repository slug"
//	@Param			request	body		CreatePRRequest	true	"Pull request data"
//	@Success		201		{object}	PRResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/pulls [post]
func (h *Handler) CreatePR(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	var req CreatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pr, err := h.service.CreatePR(c.Request.Context(), repo.ID, userID, &req)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pr.ToResponse())
}

// GetPR retrieves a pull request.
//
//	@Summary		Get pull request
//	@Description	Get details of a specific pull request
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string	true	"Owner ID"
//	@Param			repo	path		string	true	"Repository slug"
//	@Param			number	path		int		true	"PR number"
//	@Success		200		{object}	PRResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/pulls/{number} [get]
func (h *Handler) GetPR(c *gin.Context) {
	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pr number"})
		return
	}

	// Check read access
	userID := getUserIDOptional(c)
	canAccess, err := h.service.CanAccess(c.Request.Context(), repo.ID, userID, PermissionRead)
	if err != nil || !canAccess {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository_not_found"})
		return
	}

	pr, err := h.service.GetPR(c.Request.Context(), repo.ID, number)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr.ToResponse())
}

// ListPRs lists pull requests.
//
//	@Summary		List pull requests
//	@Description	Get all pull requests for a repository
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string	true	"Owner ID"
//	@Param			repo	path		string	true	"Repository slug"
//	@Param			status	query		string	false	"Filter by status"
//	@Param			limit	query		int		false	"Limit"		default(20)
//	@Param			offset	query		int		false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/pulls [get]
func (h *Handler) ListPRs(c *gin.Context) {
	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	// Check read access
	userID := getUserIDOptional(c)
	canAccess, err := h.service.CanAccess(c.Request.Context(), repo.ID, userID, PermissionRead)
	if err != nil || !canAccess {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository_not_found"})
		return
	}

	var status *PRStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := PRStatus(statusStr)
		status = &s
	}

	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(c.Query("offset")); err == nil && o >= 0 {
		offset = o
	}

	prs, total, err := h.service.ListPRs(c.Request.Context(), repo.ID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list PRs"})
		return
	}

	responses := make([]*PRResponse, len(prs))
	for i, pr := range prs {
		responses[i] = pr.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"pull_requests": responses,
		"total_count":   total,
		"limit":         limit,
		"offset":        offset,
	})
}

// UpdatePR updates a pull request.
//
//	@Summary		Update pull request
//	@Description	Update a pull request's title or description
//	@Tags			Git
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string			true	"Owner ID"
//	@Param			repo	path		string			true	"Repository slug"
//	@Param			number	path		int				true	"PR number"
//	@Param			request	body		UpdatePRRequest	true	"Update data"
//	@Success		200		{object}	PRResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/pulls/{number} [patch]
func (h *Handler) UpdatePR(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pr number"})
		return
	}

	var req UpdatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pr, err := h.service.UpdatePR(c.Request.Context(), repo.ID, number, userID, &req)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr.ToResponse())
}

// MergePR merges a pull request.
//
//	@Summary		Merge pull request
//	@Description	Merge a pull request into the base branch
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string	true	"Owner ID"
//	@Param			repo	path		string	true	"Repository slug"
//	@Param			number	path		int		true	"PR number"
//	@Success		200		{object}	PRResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/pulls/{number}/merge [post]
func (h *Handler) MergePR(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pr number"})
		return
	}

	pr, err := h.service.MergePR(c.Request.Context(), repo.ID, number, userID)
	if err != nil {
		handleGitError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr.ToResponse())
}

// --- Storage Handlers ---

// GetStorageStats returns storage statistics for a repository.
//
//	@Summary		Get repository storage stats
//	@Description	Get storage usage statistics for a repository
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner	path		string	true	"Owner ID"
//	@Param			repo	path		string	true	"Repository slug"
//	@Success		200		{object}	StorageStats
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/repos/{owner}/{repo}/storage [get]
func (h *Handler) GetStorageStats(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, repo, err := h.resolveRepo(c)
	if err != nil {
		return
	}

	// Must have read access
	if err := h.service.CheckAccess(c.Request.Context(), repo.ID, userID, PermissionRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access_denied"})
		return
	}

	stats, err := h.service.GetStorageStats(c.Request.Context(), repo.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get storage stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserStorageStats returns storage statistics for the user.
//
//	@Summary		Get user storage stats
//	@Description	Get total storage usage for the current user
//	@Tags			Git
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserStorageStats
//	@Failure		401	{object}	map[string]string
//	@Router			/storage [get]
func (h *Handler) GetUserStorageStats(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	stats, err := h.service.GetUserStorageStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get storage stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// --- Helpers ---

func getUserID(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

func getUserIDOptional(c *gin.Context) *uuid.UUID {
	id := getUserID(c)
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// resolveRepo resolves a repository from URL parameters (owner/repo).
func (h *Handler) resolveRepo(c *gin.Context) (uuid.UUID, *GitRepo, error) {
	ownerParam := c.Param("owner")
	repoSlug := c.Param("repo")

	// Owner is user ID
	ownerID, err := uuid.Parse(ownerParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner"})
		return uuid.Nil, nil, err
	}

	repo, err := h.service.GetRepoByOwnerAndSlug(c.Request.Context(), ownerID, repoSlug)
	if err != nil {
		handleGitError(c, err)
		return uuid.Nil, nil, err
	}

	return ownerID, repo, nil
}

func parseRepoFilter(c *gin.Context) *RepoFilter {
	filter := &RepoFilter{
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		filter.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil && pageSize > 0 && pageSize <= 100 {
		filter.PageSize = pageSize
	}
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}
	if typeStr := c.Query("type"); typeStr != "" {
		t := RepoType(typeStr)
		filter.Type = &t
	}
	if visibility := c.Query("visibility"); visibility != "" {
		v := Visibility(visibility)
		filter.Visibility = &v
	}

	return filter
}

func handleGitError(c *gin.Context, err error) {
	switch {
	// Repository errors
	case errors.Is(err, ErrRepoNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "repository_not_found"})
	case errors.Is(err, ErrRepoAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "repository_already_exists"})
	case errors.Is(err, ErrInvalidRepoName):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_repository_name"})
	case errors.Is(err, ErrStorageQuotaExceeded):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "storage_quota_exceeded"})

	// Access errors
	case errors.Is(err, ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access_denied"})
	case errors.Is(err, ErrNotOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": "not_owner"})
	case errors.Is(err, ErrNotCollaborator):
		c.JSON(http.StatusNotFound, gin.H{"error": "collaborator_not_found"})
	case errors.Is(err, ErrInvalidPermission):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_permission"})

	// LFS errors
	case errors.Is(err, ErrLFSNotEnabled):
		c.JSON(http.StatusBadRequest, gin.H{"error": "lfs_not_enabled"})
	case errors.Is(err, ErrLFSFileTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file_too_large"})
	case errors.Is(err, ErrLFSQuotaExceeded):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "lfs_quota_exceeded"})

	// Lock errors
	case errors.Is(err, ErrLockNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "lock_not_found"})
	case errors.Is(err, ErrLockAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "file_already_locked"})
	case errors.Is(err, ErrLockNotOwned):
		c.JSON(http.StatusForbidden, gin.H{"error": "lock_owned_by_another_user"})

	// PR errors
	case errors.Is(err, ErrPRNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "pull_request_not_found"})
	case errors.Is(err, ErrPRAlreadyMerged):
		c.JSON(http.StatusBadRequest, gin.H{"error": "pull_request_already_merged"})
	case errors.Is(err, ErrPRAlreadyClosed):
		c.JSON(http.StatusBadRequest, gin.H{"error": "pull_request_already_closed"})
	case errors.Is(err, ErrSameBranch):
		c.JSON(http.StatusBadRequest, gin.H{"error": "same_branch"})
	case errors.Is(err, ErrInvalidBranch):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_branch"})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
