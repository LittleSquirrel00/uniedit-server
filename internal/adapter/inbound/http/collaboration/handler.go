package collabhttp

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// Handler handles collaboration HTTP requests.
type Handler struct {
	domain  *collaboration.Domain
	baseURL string
}

// NewHandler creates a new collaboration handler.
func NewHandler(domain *collaboration.Domain, baseURL string) *Handler {
	return &Handler{
		domain:  domain,
		baseURL: baseURL,
	}
}

// RegisterRoutes registers collaboration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	teams := r.Group("/teams")
	teams.Use(authMiddleware)
	{
		teams.POST("", h.CreateTeam)
		teams.GET("", h.ListMyTeams)
		teams.GET("/:slug", h.GetTeam)
		teams.PATCH("/:slug", h.UpdateTeam)
		teams.DELETE("/:slug", h.DeleteTeam)

		// Members
		teams.GET("/:slug/members", h.ListMembers)
		teams.PATCH("/:slug/members/:user_id", h.UpdateMemberRole)
		teams.DELETE("/:slug/members/:user_id", h.RemoveMember)
		teams.POST("/:slug/leave", h.LeaveTeam)

		// Team invitations
		teams.POST("/:slug/invitations", h.SendInvitation)
		teams.GET("/:slug/invitations", h.ListTeamInvitations)
	}

	invitations := r.Group("/invitations")
	invitations.Use(authMiddleware)
	{
		invitations.GET("", h.ListMyInvitations)
		invitations.POST("/:token/accept", h.AcceptInvitation)
		invitations.POST("/:token/reject", h.RejectInvitation)
		invitations.DELETE("/:id", h.RevokeInvitation)
	}
}

// ========== Team Handlers ==========

// CreateTeam handles team creation.
func (h *Handler) CreateTeam(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input inbound.CreateTeamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.domain.CreateTeam(c.Request.Context(), userID, &input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, output)
}

// GetTeam handles getting a team.
func (h *Handler) GetTeam(c *gin.Context) {
	userID := getUserID(c)
	ownerID := h.getOwnerID(c, userID)

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
		return
	}

	var requesterID *uuid.UUID
	if userID != uuid.Nil {
		requesterID = &userID
	}

	output, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, requesterID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}

// ListMyTeams handles listing user's teams.
func (h *Handler) ListMyTeams(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := getQueryInt(c, "limit", 20)
	offset := getQueryInt(c, "offset", 0)

	teams, err := h.domain.ListMyTeams(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// UpdateTeam handles updating a team.
func (h *Handler) UpdateTeam(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")

	// Get team first
	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var input inbound.UpdateTeamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.domain.UpdateTeam(c.Request.Context(), team.ID, userID, &input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}

// DeleteTeam handles deleting a team.
func (h *Handler) DeleteTeam(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")

	// Get team first
	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.domain.DeleteTeam(c.Request.Context(), team.ID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
}

// ========== Member Handlers ==========

// ListMembers handles listing team members.
func (h *Handler) ListMembers(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")

	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	members, err := h.domain.ListMembers(c.Request.Context(), team.ID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// UpdateMemberRole handles updating a member's role.
func (h *Handler) UpdateMemberRole(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")
	targetUserIDStr := c.Param("user_id")

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var input inbound.UpdateMemberRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.domain.UpdateMemberRole(c.Request.Context(), team.ID, targetUserID, userID, input.Role); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member role updated"})
}

// RemoveMember handles removing a member.
func (h *Handler) RemoveMember(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")
	targetUserIDStr := c.Param("user_id")

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.domain.RemoveMember(c.Request.Context(), team.ID, targetUserID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// LeaveTeam handles leaving a team.
func (h *Handler) LeaveTeam(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")

	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.domain.LeaveTeam(c.Request.Context(), team.ID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left team successfully"})
}

// ========== Invitation Handlers ==========

// SendInvitation handles sending an invitation.
func (h *Handler) SendInvitation(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")

	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var input inbound.InviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.domain.SendInvitation(c.Request.Context(), team.ID, userID, &input)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, output)
}

// ListTeamInvitations handles listing team invitations.
func (h *Handler) ListTeamInvitations(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ownerID := h.getOwnerID(c, userID)
	slug := c.Param("slug")

	team, err := h.domain.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var status *model.InvitationStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := model.InvitationStatus(statusStr)
		status = &s
	}

	limit := getQueryInt(c, "limit", 20)
	offset := getQueryInt(c, "offset", 0)

	invitations, err := h.domain.ListTeamInvitations(c.Request.Context(), team.ID, userID, status, limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitations": invitations})
}

// ListMyInvitations handles listing user's pending invitations.
func (h *Handler) ListMyInvitations(c *gin.Context) {
	userEmail := getUserEmail(c)
	if userEmail == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit := getQueryInt(c, "limit", 20)
	offset := getQueryInt(c, "offset", 0)

	invitations, err := h.domain.ListMyInvitations(c.Request.Context(), userEmail, limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitations": invitations})
}

// AcceptInvitation handles accepting an invitation.
func (h *Handler) AcceptInvitation(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userEmail := getUserEmail(c)
	if userEmail == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	team, err := h.domain.AcceptInvitation(c.Request.Context(), token, userID, userEmail)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invitation accepted",
		"team":    team,
	})
}

// RejectInvitation handles rejecting an invitation.
func (h *Handler) RejectInvitation(c *gin.Context) {
	userEmail := getUserEmail(c)
	if userEmail == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if err := h.domain.RejectInvitation(c.Request.Context(), token, userEmail); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation rejected"})
}

// RevokeInvitation handles revoking an invitation.
func (h *Handler) RevokeInvitation(c *gin.Context) {
	userID := getUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	invitationIDStr := c.Param("id")
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
		return
	}

	if err := h.domain.RevokeInvitation(c.Request.Context(), invitationID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked"})
}

// ========== Helper Methods ==========

// getOwnerID extracts owner ID from query or defaults to current user.
func (h *Handler) getOwnerID(c *gin.Context, defaultID uuid.UUID) uuid.UUID {
	ownerIDStr := c.Query("owner_id")
	if ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err == nil {
			return ownerID
		}
	}
	return defaultID
}

func getUserID(c *gin.Context) uuid.UUID {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// getUserEmail extracts user email from context.
func getUserEmail(c *gin.Context) string {
	emailVal, exists := c.Get("email")
	if !exists {
		return ""
	}
	email, ok := emailVal.(string)
	if !ok {
		return ""
	}
	return email
}

// getQueryInt extracts an integer query parameter with a default value.
func getQueryInt(c *gin.Context, key string, defaultVal int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// handleError handles collaboration domain errors.
func (h *Handler) handleError(c *gin.Context, err error) {
	switch err {
	case collaboration.ErrTeamNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "team_not_found"})
	case collaboration.ErrSlugAlreadyExists:
		c.JSON(http.StatusConflict, gin.H{"error": "slug_already_exists"})
	case collaboration.ErrMemberNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "member_not_found"})
	case collaboration.ErrAlreadyMember:
		c.JSON(http.StatusConflict, gin.H{"error": "user_already_member"})
	case collaboration.ErrMemberLimitExceeded:
		c.JSON(http.StatusForbidden, gin.H{"error": "member_limit_exceeded"})
	case collaboration.ErrCannotChangeOwner:
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot_change_owner_role"})
	case collaboration.ErrCannotRemoveOwner:
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot_remove_owner"})
	case collaboration.ErrOnlyOwnerCanDelete:
		c.JSON(http.StatusForbidden, gin.H{"error": "only_owner_can_delete"})
	case collaboration.ErrOnlyOwnerCanTransfer:
		c.JSON(http.StatusForbidden, gin.H{"error": "only_owner_can_transfer"})
	case collaboration.ErrInsufficientPermission:
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient_permission"})
	case collaboration.ErrInvalidRole:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_role"})
	case collaboration.ErrInvitationNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation_not_found"})
	case collaboration.ErrInvitationExpired:
		c.JSON(http.StatusGone, gin.H{"error": "invitation_expired"})
	case collaboration.ErrInvitationAlreadyProcessed:
		c.JSON(http.StatusGone, gin.H{"error": "invitation_already_processed"})
	case collaboration.ErrInvitationAlreadyPending:
		c.JSON(http.StatusConflict, gin.H{"error": "invitation_already_pending"})
	case collaboration.ErrInvitationNotForYou:
		c.JSON(http.StatusForbidden, gin.H{"error": "invitation_not_for_you"})
	case collaboration.ErrCannotRevokeProcessed:
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot_revoke_processed_invitation"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

// Compile-time interface check
var _ inbound.CollaborationHttpPort = (*Handler)(nil)
