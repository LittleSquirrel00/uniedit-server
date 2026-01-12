package collaborationproto

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	collabv1 "github.com/uniedit/server/api/pb/collaboration"
	commonv1 "github.com/uniedit/server/api/pb/common"
	collabhttp "github.com/uniedit/server/internal/adapter/inbound/http/collaboration"
	"github.com/uniedit/server/internal/transport/protohttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Handler adapts collaboration HTTP handlers to proto-defined interfaces.
type Handler struct {
	collab *collabhttp.Handler
}

// NewHandler creates a new collaboration proto adapter.
func NewHandler(collab *collabhttp.Handler) *Handler {
	return &Handler{collab: collab}
}

func (h *Handler) CreateTeam(c *gin.Context, in *collabv1.CreateTeamRequest) (*collabv1.Team, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.collab.CreateTeam(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListMyTeams(c *gin.Context, in *collabv1.ListMyTeamsRequest) (*collabv1.ListMyTeamsResponse, error) {
	h.collab.ListMyTeams(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) GetTeam(c *gin.Context, in *collabv1.GetTeamRequest) (*collabv1.Team, error) {
	h.collab.GetTeam(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdateTeam(c *gin.Context, in *collabv1.UpdateTeamRequest) (*collabv1.Team, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.collab.UpdateTeam(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) DeleteTeam(c *gin.Context, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error) {
	h.collab.DeleteTeam(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListMembers(c *gin.Context, in *collabv1.GetTeamRequest) (*collabv1.ListMembersResponse, error) {
	h.collab.ListMembers(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) UpdateMemberRole(c *gin.Context, in *collabv1.UpdateMemberRoleRequest) (*commonv1.MessageResponse, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.collab.UpdateMemberRole(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) RemoveMember(c *gin.Context, in *collabv1.RemoveMemberRequest) (*commonv1.MessageResponse, error) {
	h.collab.RemoveMember(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) LeaveTeam(c *gin.Context, in *collabv1.GetTeamRequest) (*commonv1.MessageResponse, error) {
	h.collab.LeaveTeam(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) SendInvitation(c *gin.Context, in *collabv1.SendInvitationRequest) (*collabv1.Invitation, error) {
	if err := resetBody(c, in); err != nil {
		return nil, err
	}
	h.collab.SendInvitation(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListTeamInvitations(c *gin.Context, in *collabv1.ListTeamInvitationsRequest) (*collabv1.ListTeamInvitationsResponse, error) {
	h.collab.ListTeamInvitations(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) ListMyInvitations(c *gin.Context, in *collabv1.ListMyInvitationsRequest) (*collabv1.ListMyInvitationsResponse, error) {
	h.collab.ListMyInvitations(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) AcceptInvitation(c *gin.Context, in *collabv1.InvitationTokenRequest) (*collabv1.AcceptInvitationResponse, error) {
	h.collab.AcceptInvitation(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) RejectInvitation(c *gin.Context, in *collabv1.InvitationTokenRequest) (*commonv1.MessageResponse, error) {
	h.collab.RejectInvitation(c)
	return nil, protohttp.ErrHandled
}

func (h *Handler) RevokeInvitation(c *gin.Context, in *collabv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	h.collab.RevokeInvitation(c)
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
