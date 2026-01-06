package lfs

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Lock represents an LFS file lock.
type Lock struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	LockedAt time.Time `json:"locked_at"`
	Owner    LockOwner `json:"owner"`
}

// LockOwner represents the owner of a lock.
type LockOwner struct {
	Name string `json:"name"`
}

// CreateLockRequest represents a request to create a lock.
type CreateLockRequest struct {
	Path string   `json:"path"`
	Ref  *LockRef `json:"ref,omitempty"`
}

// LockRef represents a Git reference for locking.
type LockRef struct {
	Name string `json:"name"`
}

// CreateLockResponse represents a lock creation response.
type CreateLockResponse struct {
	Lock *Lock `json:"lock"`
}

// ListLocksResponse represents a lock list response.
type ListLocksResponse struct {
	Locks      []*Lock `json:"locks"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

// VerifyLocksRequest represents a request to verify locks.
type VerifyLocksRequest struct {
	Ref    *LockRef `json:"ref,omitempty"`
	Cursor string   `json:"cursor,omitempty"`
	Limit  int      `json:"limit,omitempty"`
}

// VerifyLocksResponse represents a lock verification response.
type VerifyLocksResponse struct {
	Ours       []*Lock `json:"ours"`
	Theirs     []*Lock `json:"theirs"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

// UnlockRequest represents a request to unlock a file.
type UnlockRequest struct {
	Force bool     `json:"force,omitempty"`
	Ref   *LockRef `json:"ref,omitempty"`
}

// UnlockResponse represents an unlock response.
type UnlockResponse struct {
	Lock *Lock `json:"lock"`
}

// LockRepository defines the interface for lock storage.
type LockRepository interface {
	CreateLock(ctx context.Context, repoID uuid.UUID, path string, ownerID uuid.UUID) (*LockRecord, error)
	GetLock(ctx context.Context, lockID uuid.UUID) (*LockRecord, error)
	GetLockByPath(ctx context.Context, repoID uuid.UUID, path string) (*LockRecord, error)
	ListLocks(ctx context.Context, repoID uuid.UUID, path string, cursor string, limit int) ([]*LockRecord, string, error)
	DeleteLock(ctx context.Context, lockID uuid.UUID) error
	GetUserName(ctx context.Context, userID uuid.UUID) (string, error)
}

// LockRecord represents a lock in the database.
type LockRecord struct {
	ID       uuid.UUID
	RepoID   uuid.UUID
	Path     string
	OwnerID  uuid.UUID
	LockedAt time.Time
}

// LockHandler handles LFS lock API requests.
type LockHandler struct {
	repo     LockRepository
	resolver RepoResolver
	logger   *zap.Logger
}

// NewLockHandler creates a new LFS lock handler.
func NewLockHandler(repo LockRepository, resolver RepoResolver, logger *zap.Logger) *LockHandler {
	return &LockHandler{
		repo:     repo,
		resolver: resolver,
		logger:   logger,
	}
}

// RegisterLockRoutes registers LFS lock routes.
func (h *LockHandler) RegisterLockRoutes(r *gin.RouterGroup) {
	locks := r.Group("/lfs/:owner/:repo/locks")
	{
		locks.POST("", h.CreateLock)
		locks.GET("", h.ListLocks)
		locks.POST("/verify", h.VerifyLocks)
		locks.POST("/:id/unlock", h.Unlock)
	}
}

// CreateLock creates a new file lock.
func (h *LockHandler) CreateLock(c *gin.Context) {
	h.setLFSHeaders(c)

	userID := getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required"})
		return
	}

	repoID, err := h.resolveRepo(c, true)
	if err != nil {
		return // Error already handled
	}

	var req CreateLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Path is required"})
		return
	}

	// Check if path is already locked
	existing, err := h.repo.GetLockByPath(c.Request.Context(), repoID, req.Path)
	if err == nil && existing != nil {
		ownerName, _ := h.repo.GetUserName(c.Request.Context(), existing.OwnerID)
		c.JSON(http.StatusConflict, gin.H{
			"lock": h.toLock(existing, ownerName),
			"message": "already locked",
		})
		return
	}

	// Create lock
	lock, err := h.repo.CreateLock(c.Request.Context(), repoID, req.Path, *userID)
	if err != nil {
		h.logger.Error("create lock failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create lock"})
		return
	}

	ownerName, _ := h.repo.GetUserName(c.Request.Context(), lock.OwnerID)
	c.JSON(http.StatusCreated, CreateLockResponse{
		Lock: h.toLock(lock, ownerName),
	})
}

// ListLocks lists locks for a repository.
func (h *LockHandler) ListLocks(c *gin.Context) {
	h.setLFSHeaders(c)

	repoID, err := h.resolveRepo(c, false)
	if err != nil {
		return
	}

	path := c.Query("path")
	cursor := c.Query("cursor")
	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	locks, nextCursor, err := h.repo.ListLocks(c.Request.Context(), repoID, path, cursor, limit)
	if err != nil {
		h.logger.Error("list locks failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to list locks"})
		return
	}

	response := &ListLocksResponse{
		Locks:      make([]*Lock, 0, len(locks)),
		NextCursor: nextCursor,
	}

	for _, lock := range locks {
		ownerName, _ := h.repo.GetUserName(c.Request.Context(), lock.OwnerID)
		response.Locks = append(response.Locks, h.toLock(lock, ownerName))
	}

	c.JSON(http.StatusOK, response)
}

// VerifyLocks verifies which locks are held by the current user.
func (h *LockHandler) VerifyLocks(c *gin.Context) {
	h.setLFSHeaders(c)

	userID := getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required"})
		return
	}

	repoID, err := h.resolveRepo(c, false)
	if err != nil {
		return
	}

	var req VerifyLocksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Ignore body parse errors, use defaults
	}

	limit := 100
	if req.Limit > 0 && req.Limit <= 100 {
		limit = req.Limit
	}

	locks, nextCursor, err := h.repo.ListLocks(c.Request.Context(), repoID, "", req.Cursor, limit)
	if err != nil {
		h.logger.Error("list locks failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to list locks"})
		return
	}

	response := &VerifyLocksResponse{
		Ours:       make([]*Lock, 0),
		Theirs:     make([]*Lock, 0),
		NextCursor: nextCursor,
	}

	for _, lock := range locks {
		ownerName, _ := h.repo.GetUserName(c.Request.Context(), lock.OwnerID)
		l := h.toLock(lock, ownerName)

		if lock.OwnerID == *userID {
			response.Ours = append(response.Ours, l)
		} else {
			response.Theirs = append(response.Theirs, l)
		}
	}

	c.JSON(http.StatusOK, response)
}

// Unlock removes a file lock.
func (h *LockHandler) Unlock(c *gin.Context) {
	h.setLFSHeaders(c)

	userID := getUserIDFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required"})
		return
	}

	repoID, err := h.resolveRepo(c, true)
	if err != nil {
		return
	}

	lockID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid lock ID"})
		return
	}

	var req UnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Ignore body parse errors
	}

	// Get lock
	lock, err := h.repo.GetLock(c.Request.Context(), lockID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Lock not found"})
		return
	}

	// Verify lock belongs to this repo
	if lock.RepoID != repoID {
		c.JSON(http.StatusNotFound, gin.H{"message": "Lock not found"})
		return
	}

	// Check ownership (unless force)
	if lock.OwnerID != *userID && !req.Force {
		ownerName, _ := h.repo.GetUserName(c.Request.Context(), lock.OwnerID)
		c.JSON(http.StatusForbidden, gin.H{
			"lock":    h.toLock(lock, ownerName),
			"message": "Lock is owned by another user",
		})
		return
	}

	// Delete lock
	if err := h.repo.DeleteLock(c.Request.Context(), lockID); err != nil {
		h.logger.Error("delete lock failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete lock"})
		return
	}

	ownerName, _ := h.repo.GetUserName(c.Request.Context(), lock.OwnerID)
	c.JSON(http.StatusOK, UnlockResponse{
		Lock: h.toLock(lock, ownerName),
	})
}

// resolveRepo resolves the repository from URL parameters.
func (h *LockHandler) resolveRepo(c *gin.Context, needsWrite bool) (uuid.UUID, error) {
	ownerParam := c.Param("owner")
	repoSlug := c.Param("repo")

	// Remove .git suffix if present
	repoSlug = strings.TrimSuffix(repoSlug, ".git")

	ownerID, err := uuid.Parse(ownerParam)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Repository not found"})
		return uuid.Nil, err
	}

	repoID, lfsEnabled, _, err := h.resolver.GetRepoByOwnerAndSlug(c.Request.Context(), ownerID, repoSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Repository not found"})
		return uuid.Nil, err
	}

	if !lfsEnabled {
		c.JSON(http.StatusForbidden, gin.H{"message": "LFS is not enabled for this repository"})
		return uuid.Nil, err
	}

	// Check access
	userID := getUserIDFromContext(c)
	canAccess, err := h.resolver.CanAccess(c.Request.Context(), repoID, userID, needsWrite)
	if err != nil || !canAccess {
		c.JSON(http.StatusForbidden, gin.H{"message": "Access denied"})
		return uuid.Nil, err
	}

	return repoID, nil
}

// toLock converts a LockRecord to a Lock response.
func (h *LockHandler) toLock(record *LockRecord, ownerName string) *Lock {
	if ownerName == "" {
		ownerName = record.OwnerID.String()
	}
	return &Lock{
		ID:       record.ID.String(),
		Path:     record.Path,
		LockedAt: record.LockedAt,
		Owner: LockOwner{
			Name: ownerName,
		},
	}
}

// setLFSHeaders sets standard LFS response headers.
func (h *LockHandler) setLFSHeaders(c *gin.Context) {
	c.Header("Content-Type", "application/vnd.git-lfs+json")
}
