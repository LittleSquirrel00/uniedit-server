package lfs

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BatchOperation represents the LFS batch operation type.
type BatchOperation string

const (
	BatchOperationUpload   BatchOperation = "upload"
	BatchOperationDownload BatchOperation = "download"
)

// BatchRequest represents an LFS batch API request.
type BatchRequest struct {
	Operation BatchOperation     `json:"operation"`
	Transfers []string           `json:"transfers,omitempty"`
	Ref       *BatchRef          `json:"ref,omitempty"`
	Objects   []BatchObjectInput `json:"objects"`
}

// BatchRef represents a Git reference.
type BatchRef struct {
	Name string `json:"name"`
}

// BatchObjectInput represents an object in the batch request.
type BatchObjectInput struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

// BatchResponse represents an LFS batch API response.
type BatchResponse struct {
	Transfer string              `json:"transfer,omitempty"`
	Objects  []BatchObjectOutput `json:"objects"`
}

// BatchObjectOutput represents an object in the batch response.
type BatchObjectOutput struct {
	OID           string                  `json:"oid"`
	Size          int64                   `json:"size"`
	Authenticated bool                    `json:"authenticated,omitempty"`
	Actions       *BatchObjectActions     `json:"actions,omitempty"`
	Error         *BatchObjectError       `json:"error,omitempty"`
}

// BatchObjectActions represents available actions for an object.
type BatchObjectActions struct {
	Download *DownloadAction `json:"download,omitempty"`
	Upload   *UploadAction   `json:"upload,omitempty"`
	Verify   *VerifyAction   `json:"verify,omitempty"`
}

// BatchObjectError represents an error for an object.
type BatchObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RepoResolver resolves repository information for LFS requests.
type RepoResolver interface {
	GetRepoByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (repoID uuid.UUID, lfsEnabled bool, ownerUserID uuid.UUID, err error)
	CanAccess(ctx context.Context, repoID uuid.UUID, userID *uuid.UUID, write bool) (bool, error)
	LinkLFSObject(ctx context.Context, repoID uuid.UUID, oid string, size int64, storageKey string) error
	GetLFSObject(ctx context.Context, oid string) (size int64, exists bool, err error)
}

// BatchHandler handles LFS batch API requests.
type BatchHandler struct {
	storage     *StorageManager
	resolver    RepoResolver
	baseURL     string
	logger      *zap.Logger
}

// NewBatchHandler creates a new LFS batch handler.
func NewBatchHandler(
	storage *StorageManager,
	resolver RepoResolver,
	baseURL string,
	logger *zap.Logger,
) *BatchHandler {
	return &BatchHandler{
		storage:  storage,
		resolver: resolver,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// RegisterRoutes registers LFS routes.
func (h *BatchHandler) RegisterRoutes(r *gin.RouterGroup) {
	// LFS API endpoints
	// Pattern: /lfs/:owner/:repo/objects/batch
	lfs := r.Group("/lfs")
	{
		lfs.POST("/:owner/:repo/objects/batch", h.Batch)
	}
}

// Batch handles the LFS batch API endpoint.
func (h *BatchHandler) Batch(c *gin.Context) {
	// Validate content type
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/vnd.git-lfs+json") {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"message": "Invalid content type",
		})
		return
	}

	// Set response content type
	c.Header("Content-Type", "application/vnd.git-lfs+json")

	// Parse request
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid request body",
		})
		return
	}

	// Validate operation
	if req.Operation != BatchOperationUpload && req.Operation != BatchOperationDownload {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid operation",
		})
		return
	}

	// Resolve repository
	ownerParam := c.Param("owner")
	repoSlug := c.Param("repo")

	// Remove .git suffix if present
	repoSlug = strings.TrimSuffix(repoSlug, ".git")

	ownerID, err := uuid.Parse(ownerParam)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Repository not found",
		})
		return
	}

	repoID, lfsEnabled, _, err := h.resolver.GetRepoByOwnerAndSlug(c.Request.Context(), ownerID, repoSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Repository not found",
		})
		return
	}

	if !lfsEnabled {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "LFS is not enabled for this repository",
		})
		return
	}

	// Get user ID from auth (may be nil for downloads)
	userID := getUserIDFromContext(c)

	// Check access
	needsWrite := req.Operation == BatchOperationUpload
	canAccess, err := h.resolver.CanAccess(c.Request.Context(), repoID, userID, needsWrite)
	if err != nil || !canAccess {
		if needsWrite {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Write access required",
			})
		} else {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Access denied",
			})
		}
		return
	}

	// Process objects
	response := &BatchResponse{
		Transfer: "basic", // We use basic transfer (direct to S3/R2)
		Objects:  make([]BatchObjectOutput, 0, len(req.Objects)),
	}

	for _, obj := range req.Objects {
		output := h.processObject(c.Request.Context(), repoID, &obj, req.Operation)
		response.Objects = append(response.Objects, output)
	}

	c.JSON(http.StatusOK, response)
}

// processObject processes a single object in the batch request.
func (h *BatchHandler) processObject(ctx context.Context, repoID uuid.UUID, obj *BatchObjectInput, operation BatchOperation) BatchObjectOutput {
	output := BatchObjectOutput{
		OID:           obj.OID,
		Size:          obj.Size,
		Authenticated: true,
	}

	// Validate OID format (should be SHA-256 hex string)
	if len(obj.OID) != 64 {
		output.Error = &BatchObjectError{
			Code:    422,
			Message: "Invalid OID format",
		}
		return output
	}

	switch operation {
	case BatchOperationUpload:
		// Check if object already exists
		exists, err := h.storage.ObjectExists(ctx, obj.OID)
		if err != nil {
			h.logger.Error("check object exists failed", zap.Error(err))
			output.Error = &BatchObjectError{
				Code:    500,
				Message: "Internal error",
			}
			return output
		}

		if exists {
			// Object already exists, just link it to the repo
			storageKey := h.storage.GetStorageKey(obj.OID)
			if err := h.resolver.LinkLFSObject(ctx, repoID, obj.OID, obj.Size, storageKey); err != nil {
				h.logger.Error("link object failed", zap.Error(err))
			}
			// No actions needed, object already uploaded
			return output
		}

		// Generate upload URL
		uploadAction, err := h.storage.GenerateUploadURL(ctx, obj.OID, obj.Size)
		if err != nil {
			h.logger.Error("generate upload URL failed", zap.Error(err))
			output.Error = &BatchObjectError{
				Code:    500,
				Message: "Failed to generate upload URL",
			}
			return output
		}

		output.Actions = &BatchObjectActions{
			Upload: uploadAction,
			Verify: &VerifyAction{
				Href: h.baseURL + "/lfs/verify",
			},
		}

	case BatchOperationDownload:
		// Check if object exists in our DB
		size, exists, err := h.resolver.GetLFSObject(ctx, obj.OID)
		if err != nil {
			h.logger.Error("get LFS object failed", zap.Error(err))
			output.Error = &BatchObjectError{
				Code:    500,
				Message: "Internal error",
			}
			return output
		}

		if !exists {
			output.Error = &BatchObjectError{
				Code:    404,
				Message: "Object not found",
			}
			return output
		}

		// Update size from DB record
		if size > 0 {
			output.Size = size
		}

		// Generate download URL
		downloadAction, err := h.storage.GenerateDownloadURL(ctx, obj.OID)
		if err != nil {
			h.logger.Error("generate download URL failed", zap.Error(err))
			output.Error = &BatchObjectError{
				Code:    500,
				Message: "Failed to generate download URL",
			}
			return output
		}

		output.Actions = &BatchObjectActions{
			Download: downloadAction,
		}
	}

	return output
}

// getUserIDFromContext extracts user ID from Gin context.
func getUserIDFromContext(c *gin.Context) *uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return nil
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return nil
	}
	return &userID
}
