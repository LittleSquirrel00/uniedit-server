package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	collabCmd "github.com/uniedit/server/internal/app/command/collaboration"
	collabQuery "github.com/uniedit/server/internal/app/query/collaboration"
	"github.com/uniedit/server/internal/domain/collaboration"
)

// CollaborationHandler handles HTTP requests for collaboration using CQRS pattern.
type CollaborationHandler struct {
	// Commands
	createTeam       *collabCmd.CreateTeamHandler
	updateTeam       *collabCmd.UpdateTeamHandler
	deleteTeam       *collabCmd.DeleteTeamHandler
	sendInvitation   *collabCmd.SendInvitationHandler
	acceptInvitation *collabCmd.AcceptInvitationHandler
	rejectInvitation *collabCmd.RejectInvitationHandler
	revokeInvitation *collabCmd.RevokeInvitationHandler
	updateMemberRole *collabCmd.UpdateMemberRoleHandler
	removeMember     *collabCmd.RemoveMemberHandler
	leaveTeam        *collabCmd.LeaveTeamHandler
	// Queries
	getTeam           *collabQuery.GetTeamHandler
	listMyTeams       *collabQuery.ListMyTeamsHandler
	listMembers       *collabQuery.ListMembersHandler
	listInvitations   *collabQuery.ListInvitationsHandler
	listMyInvitations *collabQuery.ListMyInvitationsHandler
	// Base URL for invitation links
	baseURL string
}

// NewCollaborationHandler creates a new collaboration handler.
func NewCollaborationHandler(
	createTeam *collabCmd.CreateTeamHandler,
	updateTeam *collabCmd.UpdateTeamHandler,
	deleteTeam *collabCmd.DeleteTeamHandler,
	sendInvitation *collabCmd.SendInvitationHandler,
	acceptInvitation *collabCmd.AcceptInvitationHandler,
	rejectInvitation *collabCmd.RejectInvitationHandler,
	revokeInvitation *collabCmd.RevokeInvitationHandler,
	updateMemberRole *collabCmd.UpdateMemberRoleHandler,
	removeMember *collabCmd.RemoveMemberHandler,
	leaveTeam *collabCmd.LeaveTeamHandler,
	getTeam *collabQuery.GetTeamHandler,
	listMyTeams *collabQuery.ListMyTeamsHandler,
	listMembers *collabQuery.ListMembersHandler,
	listInvitations *collabQuery.ListInvitationsHandler,
	listMyInvitations *collabQuery.ListMyInvitationsHandler,
	baseURL string,
) *CollaborationHandler {
	return &CollaborationHandler{
		createTeam:        createTeam,
		updateTeam:        updateTeam,
		deleteTeam:        deleteTeam,
		sendInvitation:    sendInvitation,
		acceptInvitation:  acceptInvitation,
		rejectInvitation:  rejectInvitation,
		revokeInvitation:  revokeInvitation,
		updateMemberRole:  updateMemberRole,
		removeMember:      removeMember,
		leaveTeam:         leaveTeam,
		getTeam:           getTeam,
		listMyTeams:       listMyTeams,
		listMembers:       listMembers,
		listInvitations:   listInvitations,
		listMyInvitations: listMyInvitations,
		baseURL:           baseURL,
	}
}

// RegisterRoutes registers public collaboration routes (none currently).
func (h *CollaborationHandler) RegisterRoutes(r *gin.RouterGroup) {
	// No public routes for collaboration
}

// RegisterProtectedRoutes registers collaboration routes that require authentication.
func (h *CollaborationHandler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	teams := r.Group("/teams")
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
	{
		invitations.GET("", h.ListMyInvitations)
		invitations.POST("/:token/accept", h.AcceptInvitation)
		invitations.POST("/:token/reject", h.RejectInvitation)
		invitations.DELETE("/:id", h.RevokeInvitation)
	}
}

// --- Request Types ---

type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

type UpdateTeamRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
}

type SendInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type ListTeamsRequest struct {
	Limit  int `form:"limit,default=20"`
	Offset int `form:"offset,default=0"`
}

type ListInvitationsRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}

// --- Team Handlers ---

// CreateTeam handles team creation.
// @Summary Create team
// @Tags Collaboration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTeamRequest true "Create team request"
// @Success 201 {object} collabCmd.TeamDTO
// @Router /api/v1/teams [post]
func (h *CollaborationHandler) CreateTeam(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.createTeam.Handle(c.Request.Context(), collabCmd.CreateTeamCommand{
		OwnerID:     userID,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondCreated(c, result.Team)
}

// GetTeam handles getting a team.
// @Summary Get team
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Param owner_id query string false "Owner ID"
// @Success 200 {object} collabQuery.TeamDTO
// @Router /api/v1/teams/{slug} [get]
func (h *CollaborationHandler) GetTeam(c *gin.Context) {
	var requesterID *uuid.UUID
	if uid := getUserIDOrNil(c); uid != uuid.Nil {
		requesterID = &uid
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "slug is required")
		return
	}

	result, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: requesterID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, result.Team)
}

// ListMyTeams handles listing teams the user belongs to.
// @Summary List my teams
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/teams [get]
func (h *CollaborationHandler) ListMyTeams(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	var req ListTeamsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	teams, err := h.listMyTeams.Handle(c.Request.Context(), collabQuery.ListMyTeamsQuery{
		UserID: userID,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"teams": teams})
}

// UpdateTeam handles updating a team.
// @Summary Update team
// @Tags Collaboration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Param request body UpdateTeamRequest true "Update request"
// @Success 200 {object} collabCmd.TeamDTO
// @Router /api/v1/teams/{slug} [patch]
func (h *CollaborationHandler) UpdateTeam(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")

	// Get team first to get team ID
	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.updateTeam.Handle(c.Request.Context(), collabCmd.UpdateTeamCommand{
		TeamID:      teamID,
		RequesterID: userID,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, result.Team)
}

// DeleteTeam handles deleting a team.
// @Summary Delete team
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Success 200 {object} map[string]string
// @Router /api/v1/teams/{slug} [delete]
func (h *CollaborationHandler) DeleteTeam(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")

	// Get team first
	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	_, err = h.deleteTeam.Handle(c.Request.Context(), collabCmd.DeleteTeamCommand{
		TeamID:      teamID,
		RequesterID: userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Team deleted successfully"})
}

// --- Member Handlers ---

// ListMembers handles listing team members.
// @Summary List team members
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/teams/{slug}/members [get]
func (h *CollaborationHandler) ListMembers(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")

	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	members, err := h.listMembers.Handle(c.Request.Context(), collabQuery.ListMembersQuery{
		TeamID:      teamID,
		RequesterID: userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"members": members})
}

// UpdateMemberRole handles updating a member's role.
// @Summary Update member role
// @Tags Collaboration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Param user_id path string true "User ID"
// @Param request body UpdateMemberRoleRequest true "Role update"
// @Success 200 {object} map[string]string
// @Router /api/v1/teams/{slug}/members/{user_id} [patch]
func (h *CollaborationHandler) UpdateMemberRole(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")
	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "invalid user_id")
		return
	}

	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_, err = h.updateMemberRole.Handle(c.Request.Context(), collabCmd.UpdateMemberRoleCommand{
		TeamID:       teamID,
		TargetUserID: targetUserID,
		RequesterID:  userID,
		NewRole:      req.Role,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Member role updated"})
}

// RemoveMember handles removing a member.
// @Summary Remove member
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/teams/{slug}/members/{user_id} [delete]
func (h *CollaborationHandler) RemoveMember(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")
	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "invalid user_id")
		return
	}

	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	_, err = h.removeMember.Handle(c.Request.Context(), collabCmd.RemoveMemberCommand{
		TeamID:       teamID,
		TargetUserID: targetUserID,
		RequesterID:  userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Member removed"})
}

// LeaveTeam handles leaving a team.
// @Summary Leave team
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Success 200 {object} map[string]string
// @Router /api/v1/teams/{slug}/leave [post]
func (h *CollaborationHandler) LeaveTeam(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")

	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	_, err = h.leaveTeam.Handle(c.Request.Context(), collabCmd.LeaveTeamCommand{
		TeamID: teamID,
		UserID: userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Left team successfully"})
}

// --- Invitation Handlers ---

// SendInvitation handles sending an invitation.
// @Summary Send invitation
// @Tags Collaboration
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Param request body SendInvitationRequest true "Invitation request"
// @Success 201 {object} collabCmd.InvitationDTO
// @Router /api/v1/teams/{slug}/invitations [post]
func (h *CollaborationHandler) SendInvitation(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")

	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	var req SendInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.sendInvitation.Handle(c.Request.Context(), collabCmd.SendInvitationCommand{
		TeamID:    teamID,
		InviterID: userID,
		Email:     req.Email,
		Role:      req.Role,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondCreated(c, result.Invitation)
}

// ListTeamInvitations handles listing team invitations.
// @Summary List team invitations
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param slug path string true "Team slug"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/teams/{slug}/invitations [get]
func (h *CollaborationHandler) ListTeamInvitations(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	ownerID := h.getOwnerID(c)
	if ownerID == uuid.Nil {
		return
	}

	slug := c.Param("slug")

	teamResult, err := h.getTeam.Handle(c.Request.Context(), collabQuery.GetTeamQuery{
		OwnerID:     ownerID,
		Slug:        slug,
		RequesterID: &userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	teamID, err := uuid.Parse(teamResult.Team.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "invalid team id")
		return
	}

	var req ListInvitationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var status *string
	if req.Status != "" {
		status = &req.Status
	}

	invitations, err := h.listInvitations.Handle(c.Request.Context(), collabQuery.ListInvitationsQuery{
		TeamID:      teamID,
		RequesterID: userID,
		Status:      status,
		Limit:       req.Limit,
		Offset:      req.Offset,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"invitations": invitations})
}

// ListMyInvitations handles listing user's pending invitations.
// @Summary List my invitations
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/invitations [get]
func (h *CollaborationHandler) ListMyInvitations(c *gin.Context) {
	userEmail := getUserEmail(c)
	if userEmail == "" {
		respondError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req ListTeamsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	invitations, err := h.listMyInvitations.Handle(c.Request.Context(), collabQuery.ListMyInvitationsQuery{
		Email:  userEmail,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"invitations": invitations})
}

// AcceptInvitation handles accepting an invitation.
// @Summary Accept invitation
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param token path string true "Invitation token"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/invitations/{token}/accept [post]
func (h *CollaborationHandler) AcceptInvitation(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	userEmail := getUserEmail(c)
	if userEmail == "" {
		respondError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	token := c.Param("token")
	if token == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}

	result, err := h.acceptInvitation.Handle(c.Request.Context(), collabCmd.AcceptInvitationCommand{
		Token:     token,
		UserID:    userID,
		UserEmail: userEmail,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{
		"message": "Invitation accepted",
		"team":    result.Team,
	})
}

// RejectInvitation handles rejecting an invitation.
// @Summary Reject invitation
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param token path string true "Invitation token"
// @Success 200 {object} map[string]string
// @Router /api/v1/invitations/{token}/reject [post]
func (h *CollaborationHandler) RejectInvitation(c *gin.Context) {
	userEmail := getUserEmail(c)
	if userEmail == "" {
		respondError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	token := c.Param("token")
	if token == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}

	_, err := h.rejectInvitation.Handle(c.Request.Context(), collabCmd.RejectInvitationCommand{
		Token:     token,
		UserEmail: userEmail,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Invitation rejected"})
}

// RevokeInvitation handles revoking an invitation.
// @Summary Revoke invitation
// @Tags Collaboration
// @Produce json
// @Security BearerAuth
// @Param id path string true "Invitation ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/invitations/{id} [delete]
func (h *CollaborationHandler) RevokeInvitation(c *gin.Context) {
	userID := requireAuth(c)
	if userID == uuid.Nil {
		return
	}

	invitationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "invalid invitation id")
		return
	}

	_, err = h.revokeInvitation.Handle(c.Request.Context(), collabCmd.RevokeInvitationCommand{
		InvitationID: invitationID,
		RequesterID:  userID,
	})
	if err != nil {
		h.handleCollaborationError(c, err)
		return
	}

	respondSuccess(c, gin.H{"message": "Invitation revoked"})
}

// --- Helper Methods ---

func (h *CollaborationHandler) getOwnerID(c *gin.Context) uuid.UUID {
	// First check if owner is specified in query
	ownerIDStr := c.Query("owner_id")
	if ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_request", "invalid owner_id")
			return uuid.Nil
		}
		return ownerID
	}

	// Default to current user
	return requireAuth(c)
}

func getUserIDOrNil(c *gin.Context) uuid.UUID {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}

	if userID, ok := userIDVal.(uuid.UUID); ok {
		return userID
	}

	if idStr, ok := userIDVal.(string); ok {
		if userID, err := uuid.Parse(idStr); err == nil {
			return userID
		}
	}

	return uuid.Nil
}

func getUserEmail(c *gin.Context) string {
	emailVal, exists := c.Get("user_email")
	if !exists {
		return ""
	}

	if email, ok := emailVal.(string); ok {
		return email
	}

	return ""
}

func (h *CollaborationHandler) handleCollaborationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, collaboration.ErrTeamNotFound):
		respondError(c, http.StatusNotFound, "team_not_found", "Team not found")
	case errors.Is(err, collaboration.ErrSlugAlreadyExists):
		respondError(c, http.StatusConflict, "slug_already_exists", "Team slug already exists")
	case errors.Is(err, collaboration.ErrMemberNotFound):
		respondError(c, http.StatusNotFound, "member_not_found", "Member not found")
	case errors.Is(err, collaboration.ErrAlreadyMember):
		respondError(c, http.StatusConflict, "already_member", "User is already a member")
	case errors.Is(err, collaboration.ErrMemberLimitExceeded):
		respondError(c, http.StatusForbidden, "member_limit_exceeded", "Team member limit exceeded")
	case errors.Is(err, collaboration.ErrCannotChangeOwner):
		respondError(c, http.StatusForbidden, "cannot_change_owner", "Cannot change owner role")
	case errors.Is(err, collaboration.ErrCannotRemoveOwner):
		respondError(c, http.StatusForbidden, "cannot_remove_owner", "Cannot remove team owner")
	case errors.Is(err, collaboration.ErrOnlyOwnerCanDelete):
		respondError(c, http.StatusForbidden, "only_owner_can_delete", "Only owner can delete team")
	case errors.Is(err, collaboration.ErrOnlyOwnerCanTransfer):
		respondError(c, http.StatusForbidden, "only_owner_can_transfer", "Only owner can transfer ownership")
	case errors.Is(err, collaboration.ErrInsufficientPermission):
		respondError(c, http.StatusForbidden, "insufficient_permission", "Insufficient permission")
	case errors.Is(err, collaboration.ErrInvalidRole):
		respondError(c, http.StatusBadRequest, "invalid_role", "Invalid role")
	case errors.Is(err, collaboration.ErrInvitationNotFound):
		respondError(c, http.StatusNotFound, "invitation_not_found", "Invitation not found")
	case errors.Is(err, collaboration.ErrInvitationExpired):
		respondError(c, http.StatusGone, "invitation_expired", "Invitation has expired")
	case errors.Is(err, collaboration.ErrInvitationAlreadyProcessed):
		respondError(c, http.StatusGone, "invitation_already_processed", "Invitation has already been processed")
	case errors.Is(err, collaboration.ErrInvitationAlreadyPending):
		respondError(c, http.StatusConflict, "invitation_already_pending", "Invitation already pending for this user")
	case errors.Is(err, collaboration.ErrInvitationNotForYou):
		respondError(c, http.StatusForbidden, "invitation_not_for_you", "This invitation is not for you")
	case errors.Is(err, collaboration.ErrCannotRevokeProcessed):
		respondError(c, http.StatusBadRequest, "cannot_revoke_processed", "Cannot revoke processed invitation")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An internal error occurred")
	}
}
