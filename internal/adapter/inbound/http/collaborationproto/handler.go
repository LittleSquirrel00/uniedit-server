package collaborationproto

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	collabv1 "github.com/uniedit/server/api/pb/collaboration"
	commonv1 "github.com/uniedit/server/api/pb/common"
	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	collab inbound.CollaborationDomain
}

func NewHandler(collab inbound.CollaborationDomain) *Handler {
	return &Handler{collab: collab}
}

func (h *Handler) CreateTeam(c *gin.Context, in *collabv1.CreateTeamRequest) (*collabv1.Team, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.CreateTeam(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}

	c.Status(http.StatusCreated)
	return out, nil
}

func (h *Handler) ListMyTeams(c *gin.Context, in *collabv1.ListMyTeamsRequest) (*collabv1.ListMyTeamsResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.ListMyTeams(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) GetTeam(c *gin.Context, in *collabv1.GetTeamRequest) (*collabv1.Team, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.GetTeam(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) UpdateTeam(c *gin.Context, in *collabv1.UpdateTeamRequest) (*collabv1.Team, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.UpdateTeam(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) DeleteTeam(c *gin.Context, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.DeleteTeam(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) ListMembers(c *gin.Context, in *collabv1.GetTeamRequest) (*collabv1.ListMembersResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.ListMembers(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) UpdateMemberRole(c *gin.Context, in *collabv1.UpdateMemberRoleRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.UpdateMemberRole(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) RemoveMember(c *gin.Context, in *collabv1.RemoveMemberRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.RemoveMember(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) LeaveTeam(c *gin.Context, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.LeaveTeam(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) SendInvitation(c *gin.Context, in *collabv1.SendInvitationRequest) (*collabv1.Invitation, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.SendInvitation(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}

	c.Status(http.StatusCreated)
	return out, nil
}

func (h *Handler) ListTeamInvitations(c *gin.Context, in *collabv1.ListTeamInvitationsRequest) (*collabv1.ListTeamInvitationsResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.ListTeamInvitations(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) ListMyInvitations(c *gin.Context, in *collabv1.ListMyInvitationsRequest) (*collabv1.ListMyInvitationsResponse, error) {
	email := middleware.GetEmail(c)
	if email == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.collab.ListMyInvitations(c.Request.Context(), email, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) AcceptInvitation(c *gin.Context, in *collabv1.InvitationTokenRequest) (*collabv1.AcceptInvitationResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}
	email := middleware.GetEmail(c)
	if email == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.collab.AcceptInvitation(c.Request.Context(), userID, email, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}

	return out, nil
}

func (h *Handler) RejectInvitation(c *gin.Context, in *collabv1.InvitationTokenRequest) (*commonv1.MessageResponse, error) {
	email := middleware.GetEmail(c)
	if email == "" {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.collab.RejectInvitation(c.Request.Context(), email, in)
	if err != nil {
		return nil, mapCollaborationError(err)
	}
	return out, nil
}

func (h *Handler) RevokeInvitation(c *gin.Context, in *collabv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	userID, err := requireUserID(c)
	if err != nil {
		return nil, err
	}

	out, err := h.collab.RevokeInvitation(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapCollaborationError(err)
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

func mapCollaborationError(err error) error {
	switch {
	case errors.Is(err, collaboration.ErrInvalidRequest):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: err.Error(), Err: err}
	case errors.Is(err, collaboration.ErrTeamNotFound),
		errors.Is(err, collaboration.ErrMemberNotFound),
		errors.Is(err, collaboration.ErrInvitationNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: err.Error(), Err: err}
	case errors.Is(err, collaboration.ErrSlugAlreadyExists),
		errors.Is(err, collaboration.ErrAlreadyMember),
		errors.Is(err, collaboration.ErrInvitationAlreadyPending):
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "conflict", Message: err.Error(), Err: err}
	case errors.Is(err, collaboration.ErrMemberLimitExceeded),
		errors.Is(err, collaboration.ErrCannotChangeOwner),
		errors.Is(err, collaboration.ErrCannotRemoveOwner),
		errors.Is(err, collaboration.ErrOnlyOwnerCanDelete),
		errors.Is(err, collaboration.ErrOnlyOwnerCanTransfer),
		errors.Is(err, collaboration.ErrInsufficientPermission),
		errors.Is(err, collaboration.ErrInvitationNotForYou):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: err.Error(), Err: err}
	case errors.Is(err, collaboration.ErrInvalidRole):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_role", Message: err.Error(), Err: err}
	case errors.Is(err, collaboration.ErrInvitationExpired),
		errors.Is(err, collaboration.ErrInvitationAlreadyProcessed),
		errors.Is(err, collaboration.ErrTeamDeleted):
		return &protohttp.HTTPError{Status: http.StatusGone, Code: "gone", Message: err.Error(), Err: err}
	case errors.Is(err, collaboration.ErrCannotRevokeProcessed):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: err.Error(), Err: err}
	default:
		return &protohttp.HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Err: err}
	}
}
