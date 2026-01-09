package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// CollaborationHandler handles collaboration HTTP requests.
type CollaborationHandler struct {
	domain  *collaboration.Domain
	baseURL string
}

// NewCollaborationHandler creates a new collaboration handler.
func NewCollaborationHandler(domain *collaboration.Domain, baseURL string) *CollaborationHandler {
	return &CollaborationHandler{
		domain:  domain,
		baseURL: baseURL,
	}
}

// RegisterRoutes registers collaboration routes.
func (h *CollaborationHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
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
// @Summary Create team
// @Tags Collaboration
// @Accept json
// @Produce json
// @Param request body inbound.CreateTeamInput true "Create team request"
// @Success 201 {object} inbound.TeamOutput
// @Router /teams [post]
func (h *CollaborationHandler) CreateTeam(c *gin.Context) {
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
// @Summary Get team
// @Tags Collaboration
// @Produce json
// @Param slug path string true "Team slug"
// @Param owner_id query string false "Owner ID"
// @Success 200 {object} inbound.TeamOutput
// @Router /teams/{slug} [get]
func (h *CollaborationHandler) GetTeam(c *gin.Context) {
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
// @Summary List my teams
// @Tags Collaboration
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /teams [get]
func (h *CollaborationHandler) ListMyTeams(c *gin.Context) {
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
// @Summary Update team
// @Tags Collaboration
// @Accept json
// @Produce json
// @Param slug path string true "Team slug"
// @Param request body inbound.UpdateTeamInput true "Update request"
// @Success 200 {object} inbound.TeamOutput
// @Router /teams/{slug} [patch]
func (h *CollaborationHandler) UpdateTeam(c *gin.Context) {
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
// @Summary Delete team
// @Tags Collaboration
// @Param slug path string true "Team slug"
// @Success 200 {object} map[string]string
// @Router /teams/{slug} [delete]
func (h *CollaborationHandler) DeleteTeam(c *gin.Context) {
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
// @Summary List team members
// @Tags Collaboration
// @Produce json
// @Param slug path string true "Team slug"
// @Success 200 {object} map[string]interface{}
// @Router /teams/{slug}/members [get]
func (h *CollaborationHandler) ListMembers(c *gin.Context) {
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
// @Summary Update member role
// @Tags Collaboration
// @Accept json
// @Produce json
// @Param slug path string true "Team slug"
// @Param user_id path string true "User ID"
// @Param request body inbound.UpdateMemberRoleInput true "Role update"
// @Success 200 {object} map[string]string
// @Router /teams/{slug}/members/{user_id} [patch]
func (h *CollaborationHandler) UpdateMemberRole(c *gin.Context) {
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
// @Summary Remove member
// @Tags Collaboration
// @Param slug path string true "Team slug"
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]string
// @Router /teams/{slug}/members/{user_id} [delete]
func (h *CollaborationHandler) RemoveMember(c *gin.Context) {
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
// @Summary Leave team
// @Tags Collaboration
// @Param slug path string true "Team slug"
// @Success 200 {object} map[string]string
// @Router /teams/{slug}/leave [post]
func (h *CollaborationHandler) LeaveTeam(c *gin.Context) {
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
// @Summary Send invitation
// @Tags Collaboration
// @Accept json
// @Produce json
// @Param slug path string true "Team slug"
// @Param request body inbound.InviteInput true "Invitation request"
// @Success 201 {object} inbound.InvitationOutput
// @Router /teams/{slug}/invitations [post]
func (h *CollaborationHandler) SendInvitation(c *gin.Context) {
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
// @Summary List team invitations
// @Tags Collaboration
// @Produce json
// @Param slug path string true "Team slug"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /teams/{slug}/invitations [get]
func (h *CollaborationHandler) ListTeamInvitations(c *gin.Context) {
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
// @Summary List my invitations
// @Tags Collaboration
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /invitations [get]
func (h *CollaborationHandler) ListMyInvitations(c *gin.Context) {
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
// @Summary Accept invitation
// @Tags Collaboration
// @Param token path string true "Invitation token"
// @Success 200 {object} map[string]interface{}
// @Router /invitations/{token}/accept [post]
func (h *CollaborationHandler) AcceptInvitation(c *gin.Context) {
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
// @Summary Reject invitation
// @Tags Collaboration
// @Param token path string true "Invitation token"
// @Success 200 {object} map[string]string
// @Router /invitations/{token}/reject [post]
func (h *CollaborationHandler) RejectInvitation(c *gin.Context) {
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
// @Summary Revoke invitation
// @Tags Collaboration
// @Param id path string true "Invitation ID"
// @Success 200 {object} map[string]string
// @Router /invitations/{id} [delete]
func (h *CollaborationHandler) RevokeInvitation(c *gin.Context) {
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
func (h *CollaborationHandler) getOwnerID(c *gin.Context, defaultID uuid.UUID) uuid.UUID {
	ownerIDStr := c.Query("owner_id")
	if ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err == nil {
			return ownerID
		}
	}
	return defaultID
}

// getUserEmail extracts user email from context.
func getUserEmail(c *gin.Context) string {
	emailVal, exists := c.Get("user_email")
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
func (h *CollaborationHandler) handleError(c *gin.Context, err error) {
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
var _ inbound.CollaborationHttpPort = (*CollaborationHandler)(nil)
