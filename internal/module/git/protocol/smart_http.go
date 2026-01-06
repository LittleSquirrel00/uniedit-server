package protocol

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SmartHTTPHandler handles Git Smart HTTP protocol.
type SmartHTTPHandler struct {
	logger *zap.Logger
}

// NewSmartHTTPHandler creates a new Smart HTTP handler.
func NewSmartHTTPHandler(logger *zap.Logger) *SmartHTTPHandler {
	return &SmartHTTPHandler{
		logger: logger,
	}
}

// ServiceType represents the Git service type.
type ServiceType string

const (
	ServiceUploadPack  ServiceType = "git-upload-pack"
	ServiceReceivePack ServiceType = "git-receive-pack"
)

// RequestContext contains context for a Git request.
type RequestContext struct {
	RepoID     uuid.UUID
	UserID     *uuid.UUID
	CanRead    bool
	CanWrite   bool
	Filesystem billy.Filesystem
}

// InfoRefs handles the info/refs discovery request.
func (h *SmartHTTPHandler) InfoRefs(ctx context.Context, reqCtx *RequestContext, service ServiceType) ([]byte, string, error) {
	// Validate service
	if service != ServiceUploadPack && service != ServiceReceivePack {
		return nil, "", fmt.Errorf("invalid service: %s", service)
	}

	// Check permissions
	if service == ServiceReceivePack && !reqCtx.CanWrite {
		return nil, "", transport.ErrAuthorizationFailed
	}
	if !reqCtx.CanRead {
		return nil, "", transport.ErrAuthorizationFailed
	}

	// Open repository
	storage := filesystem.NewStorage(reqCtx.Filesystem, nil)
	repo, err := git.Open(storage, nil)
	if err != nil {
		return nil, "", fmt.Errorf("open repo: %w", err)
	}

	// Build response
	var buf bytes.Buffer
	enc := pktline.NewEncoder(&buf)

	// First line includes service announcement
	serviceLine := fmt.Sprintf("# service=%s\n", service)
	if err := enc.Encode([]byte(serviceLine)); err != nil {
		return nil, "", fmt.Errorf("encode service: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, "", fmt.Errorf("flush: %w", err)
	}

	// Get references
	refs, err := repo.References()
	if err != nil {
		return nil, "", fmt.Errorf("get refs: %w", err)
	}

	// Get HEAD
	head, err := repo.Head()
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return nil, "", fmt.Errorf("get HEAD: %w", err)
	}

	// Build capabilities
	caps := []string{
		"multi_ack",
		"thin-pack",
		"side-band",
		"side-band-64k",
		"ofs-delta",
		"shallow",
		"deepen-since",
		"deepen-not",
		"deepen-relative",
		"no-progress",
		"include-tag",
		"allow-tip-sha1-in-want",
		"allow-reachable-sha1-in-want",
		"no-done",
	}
	if service == ServiceReceivePack {
		caps = append(caps, "report-status", "delete-refs", "quiet", "atomic", "push-options")
	}
	capsStr := strings.Join(caps, " ")

	// First reference line includes capabilities
	firstLine := true
	var refLines []string

	// Add HEAD if exists
	if head != nil {
		line := fmt.Sprintf("%s HEAD", head.Hash().String())
		if firstLine {
			line += "\x00" + capsStr
			firstLine = false
		}
		refLines = append(refLines, line)
	}

	// Add all references
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.SymbolicReference {
			return nil // Skip symbolic refs
		}
		line := fmt.Sprintf("%s %s", ref.Hash().String(), ref.Name())
		if firstLine {
			line += "\x00" + capsStr
			firstLine = false
		}
		refLines = append(refLines, line)
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("iterate refs: %w", err)
	}

	// If no refs, send capabilities with zero-id
	if firstLine {
		line := fmt.Sprintf("%s capabilities^{}\x00%s", plumbing.ZeroHash.String(), capsStr)
		refLines = append(refLines, line)
	}

	// Encode reference lines
	for _, line := range refLines {
		if err := enc.Encode([]byte(line + "\n")); err != nil {
			return nil, "", fmt.Errorf("encode ref: %w", err)
		}
	}
	if err := enc.Flush(); err != nil {
		return nil, "", fmt.Errorf("flush refs: %w", err)
	}

	contentType := fmt.Sprintf("application/x-%s-advertisement", service)
	return buf.Bytes(), contentType, nil
}

// UploadPack handles git-upload-pack (fetch/clone).
func (h *SmartHTTPHandler) UploadPack(ctx context.Context, reqCtx *RequestContext, r io.Reader, w io.Writer) error {
	if !reqCtx.CanRead {
		return transport.ErrAuthorizationFailed
	}

	// Decompress if gzipped
	reader := r
	if gzReader, err := gzip.NewReader(r); err == nil {
		defer gzReader.Close()
		reader = gzReader
	}

	// Open repository
	storage := filesystem.NewStorage(reqCtx.Filesystem, nil)
	repo, err := git.Open(storage, nil)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	// Parse upload-pack request
	req := packp.NewUploadPackRequest()
	if err := req.Decode(reader); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}

	h.logger.Debug("upload-pack request",
		zap.Int("wants", len(req.Wants)),
		zap.Int("haves", len(req.Haves)),
	)

	// Create upload-pack session
	session, err := h.createUploadPackSession(repo, req)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Generate response
	response, err := session.UploadPack(ctx, req)
	if err != nil {
		return fmt.Errorf("upload-pack: %w", err)
	}

	// Encode response
	if err := response.Encode(w); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	return nil
}

// ReceivePack handles git-receive-pack (push).
func (h *SmartHTTPHandler) ReceivePack(ctx context.Context, reqCtx *RequestContext, r io.Reader, w io.Writer, onPushComplete func(time.Time)) error {
	if !reqCtx.CanWrite {
		return transport.ErrAuthorizationFailed
	}

	// Decompress if gzipped
	reader := r
	if gzReader, err := gzip.NewReader(r); err == nil {
		defer gzReader.Close()
		reader = gzReader
	}

	// Open repository
	storage := filesystem.NewStorage(reqCtx.Filesystem, nil)
	repo, err := git.Open(storage, nil)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	// Parse receive-pack request
	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(reader); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}

	h.logger.Debug("receive-pack request",
		zap.Int("commands", len(req.Commands)),
	)

	// Create receive-pack session
	session, err := h.createReceivePackSession(repo, reqCtx.Filesystem)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Process reference updates
	response, err := session.ReceivePack(ctx, req)
	if err != nil {
		return fmt.Errorf("receive-pack: %w", err)
	}

	// Encode response
	if err := response.Encode(w); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	// Notify push completion
	if onPushComplete != nil {
		onPushComplete(time.Now())
	}

	return nil
}

// uploadPackSession implements upload-pack logic.
type uploadPackSession struct {
	repo *git.Repository
}

func (h *SmartHTTPHandler) createUploadPackSession(repo *git.Repository, req *packp.UploadPackRequest) (*uploadPackSession, error) {
	return &uploadPackSession{repo: repo}, nil
}

func (s *uploadPackSession) UploadPack(ctx context.Context, req *packp.UploadPackRequest) (*packp.UploadPackResponse, error) {
	// Build response
	response := packp.NewUploadPackResponse(req)

	// Get objects to send
	if len(req.Wants) == 0 {
		return response, nil
	}

	// Use go-git's pack builder
	// This is simplified - in production you'd use go-git's transport server
	encoder := s.repo.Storer
	_ = encoder // Placeholder for pack generation

	// For now, return empty packfile
	// In a full implementation, we'd generate the packfile with requested objects
	response.ACKs = make([]plumbing.Hash, 0)

	return response, nil
}

// receivePackSession implements receive-pack logic.
type receivePackSession struct {
	repo *git.Repository
	fs   billy.Filesystem
}

func (h *SmartHTTPHandler) createReceivePackSession(repo *git.Repository, fs billy.Filesystem) (*receivePackSession, error) {
	return &receivePackSession{repo: repo, fs: fs}, nil
}

func (s *receivePackSession) ReceivePack(ctx context.Context, req *packp.ReferenceUpdateRequest) (*packp.ReportStatus, error) {
	// Build status report
	status := packp.NewReportStatus()
	status.UnpackStatus = "ok"

	// Process each command
	for _, cmd := range req.Commands {
		cmdStatus := &packp.CommandStatus{
			ReferenceName: cmd.Name,
			Status:        "ok",
		}

		// Handle reference update
		switch {
		case cmd.Old == plumbing.ZeroHash:
			// Create new ref
			ref := plumbing.NewHashReference(cmd.Name, cmd.New)
			if err := s.repo.Storer.SetReference(ref); err != nil {
				cmdStatus.Status = err.Error()
			}

		case cmd.New == plumbing.ZeroHash:
			// Delete ref
			if err := s.repo.Storer.RemoveReference(cmd.Name); err != nil {
				cmdStatus.Status = err.Error()
			}

		default:
			// Update ref
			ref := plumbing.NewHashReference(cmd.Name, cmd.New)
			if err := s.repo.Storer.SetReference(ref); err != nil {
				cmdStatus.Status = err.Error()
			}
		}

		status.CommandStatuses = append(status.CommandStatuses, cmdStatus)
	}

	return status, nil
}

// GetContentType returns the appropriate content type for a Git service.
func GetContentType(service ServiceType, isRequest bool) string {
	if isRequest {
		return fmt.Sprintf("application/x-%s-request", service)
	}
	return fmt.Sprintf("application/x-%s-result", service)
}

// ParseService extracts the Git service from query parameter.
func ParseService(serviceParam string) (ServiceType, error) {
	switch serviceParam {
	case string(ServiceUploadPack):
		return ServiceUploadPack, nil
	case string(ServiceReceivePack):
		return ServiceReceivePack, nil
	default:
		return "", fmt.Errorf("unknown service: %s", serviceParam)
	}
}

// GitHTTPBackend creates a http.Handler for Git HTTP backend.
func (h *SmartHTTPHandler) GitHTTPBackend(
	resolveRepo func(r *http.Request) (*RequestContext, error),
	onPushComplete func(repoID uuid.UUID, pushedAt time.Time),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve repository context
		reqCtx, err := resolveRepo(r)
		if err != nil {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		path := r.URL.Path

		// Handle info/refs discovery
		if strings.HasSuffix(path, "/info/refs") {
			service := r.URL.Query().Get("service")
			svc, err := ParseService(service)
			if err != nil {
				http.Error(w, "Invalid service", http.StatusBadRequest)
				return
			}

			data, contentType, err := h.InfoRefs(r.Context(), reqCtx, svc)
			if err != nil {
				h.logger.Error("info/refs failed", zap.Error(err))
				if err == transport.ErrAuthorizationFailed {
					w.Header().Set("WWW-Authenticate", `Basic realm="Git"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(data)
			return
		}

		// Handle git-upload-pack
		if strings.HasSuffix(path, "/git-upload-pack") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", GetContentType(ServiceUploadPack, false))
			w.Header().Set("Cache-Control", "no-cache")

			if err := h.UploadPack(r.Context(), reqCtx, r.Body, w); err != nil {
				h.logger.Error("upload-pack failed", zap.Error(err))
				if err == transport.ErrAuthorizationFailed {
					w.Header().Set("WWW-Authenticate", `Basic realm="Git"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				return // Response may already be written
			}
			return
		}

		// Handle git-receive-pack
		if strings.HasSuffix(path, "/git-receive-pack") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", GetContentType(ServiceReceivePack, false))
			w.Header().Set("Cache-Control", "no-cache")

			onComplete := func(pushedAt time.Time) {
				if onPushComplete != nil {
					onPushComplete(reqCtx.RepoID, pushedAt)
				}
			}

			if err := h.ReceivePack(r.Context(), reqCtx, r.Body, w, onComplete); err != nil {
				h.logger.Error("receive-pack failed", zap.Error(err))
				if err == transport.ErrAuthorizationFailed {
					w.Header().Set("WWW-Authenticate", `Basic realm="Git"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				return
			}
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	})
}
