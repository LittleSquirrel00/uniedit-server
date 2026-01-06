package collaboration

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for collaboration.
type Handler struct {
	service *Service
	baseURL string
}

// NewHandler creates a new collaboration handler.
func NewHandler(service *Service, baseURL string) *Handler {
	return &Handler{
		service: service,
		baseURL: baseURL,
	}
}

// RegisterRoutes registers public collaboration routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
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

// ========== Team Handlers ==========

// CreateTeam handles team creation.
//
//	@Summary		Create team
//	@Description	Create a new team
//	@Tags			Collaboration
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateTeamRequest	true	"Create team request"
//	@Success		201		{object}	TeamResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/teams [post]
func (h *Handler) CreateTeam(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.service.CreateTeam(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	role := RoleOwner
	c.JSON(http.StatusCreated, team.ToResponse(1, &role))
}

// GetTeam handles getting a team.
//
//	@Summary		Get team
//	@Description	Get team details by slug
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug		path		string	true	"Team slug"
//	@Param			owner_id	query		string	false	"Owner ID"
//	@Success		200			{object}	TeamResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Router			/teams/{slug} [get]
func (h *Handler) GetTeam(c *gin.Context) {
	userID, _ := h.tryGetUserID(c)
	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
		return
	}

	team, myRole, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	memberCount, _ := h.service.GetMemberCount(c.Request.Context(), team.ID)
	c.JSON(http.StatusOK, team.ToResponse(memberCount, myRole))
}

// ListMyTeams handles listing teams the user belongs to.
//
//	@Summary		List my teams
//	@Description	Get all teams the current user belongs to
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Router			/teams [get]
func (h *Handler) ListMyTeams(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var query ListTeamsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	teams, err := h.service.ListMyTeams(c.Request.Context(), userID, query.Limit, query.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Build response with member counts and roles
	responses := make([]*TeamResponse, len(teams))
	for i, team := range teams {
		memberCount, _ := h.service.GetMemberCount(c.Request.Context(), team.ID)
		member, _ := h.service.repo.GetMember(c.Request.Context(), team.ID, userID)
		var myRole *Role
		if member != nil {
			myRole = &member.Role
		}
		responses[i] = team.ToResponse(memberCount, myRole)
	}

	c.JSON(http.StatusOK, gin.H{"teams": responses})
}

// UpdateTeam handles updating a team.
//
//	@Summary		Update team
//	@Description	Update team settings
//	@Tags			Collaboration
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string				true	"Team slug"
//	@Param			request	body		UpdateTeamRequest	true	"Update request"
//	@Success		200		{object}	TeamResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug} [patch]
func (h *Handler) UpdateTeam(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	// Get team first
	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedTeam, err := h.service.UpdateTeam(c.Request.Context(), team.ID, userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	memberCount, _ := h.service.GetMemberCount(c.Request.Context(), updatedTeam.ID)
	member, _ := h.service.repo.GetMember(c.Request.Context(), updatedTeam.ID, userID)
	var myRole *Role
	if member != nil {
		myRole = &member.Role
	}

	c.JSON(http.StatusOK, updatedTeam.ToResponse(memberCount, myRole))
}

// DeleteTeam handles deleting a team.
//
//	@Summary		Delete team
//	@Description	Permanently delete a team
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Team slug"
//	@Success		200		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug} [delete]
func (h *Handler) DeleteTeam(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	// Get team first
	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.service.DeleteTeam(c.Request.Context(), team.ID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
}

// ========== Member Handlers ==========

// ListMembers handles listing team members.
//
//	@Summary		List team members
//	@Description	Get all members of a team
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Team slug"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug}/members [get]
func (h *Handler) ListMembers(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	members, err := h.service.ListMembers(c.Request.Context(), team.ID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]*MemberResponse, len(members))
	for i, m := range members {
		responses[i] = &MemberResponse{
			UserID:   m.UserID,
			Email:    m.Email,
			Name:     m.Name,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"members": responses})
}

// UpdateMemberRole handles updating a member's role.
//
//	@Summary		Update member role
//	@Description	Update a team member's role
//	@Tags			Collaboration
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string					true	"Team slug"
//	@Param			user_id	path		string					true	"User ID"
//	@Param			request	body		UpdateMemberRoleRequest	true	"Role update"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug}/members/{user_id} [patch]
func (h *Handler) UpdateMemberRole(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")
	targetUserIDStr := c.Param("user_id")

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateMemberRole(c.Request.Context(), team.ID, targetUserID, userID, req.Role); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member role updated"})
}

// RemoveMember handles removing a member.
//
//	@Summary		Remove member
//	@Description	Remove a member from a team
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Team slug"
//	@Param			user_id	path		string	true	"User ID"
//	@Success		200		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug}/members/{user_id} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")
	targetUserIDStr := c.Param("user_id")

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), team.ID, targetUserID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// LeaveTeam handles leaving a team.
//
//	@Summary		Leave team
//	@Description	Leave a team (current user)
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Team slug"
//	@Success		200		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug}/leave [post]
func (h *Handler) LeaveTeam(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if err := h.service.LeaveTeam(c.Request.Context(), team.ID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left team successfully"})
}

// ========== Invitation Handlers ==========

// SendInvitation handles sending an invitation.
//
//	@Summary		Send invitation
//	@Description	Send a team invitation to a user
//	@Tags			Collaboration
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string			true	"Team slug"
//	@Param			request	body		InviteRequest	true	"Invitation request"
//	@Success		201		{object}	InvitationResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Router			/teams/{slug}/invitations [post]
func (h *Handler) SendInvitation(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var req InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invitation, err := h.service.SendInvitation(c.Request.Context(), team.ID, userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, invitation.ToResponse(true, h.baseURL))
}

// ListTeamInvitations handles listing team invitations.
//
//	@Summary		List team invitations
//	@Description	Get all invitations for a team
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Team slug"
//	@Param			status	query		string	false	"Filter by status"
//	@Param			limit	query		int		false	"Limit"		default(20)
//	@Param			offset	query		int		false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/teams/{slug}/invitations [get]
func (h *Handler) ListTeamInvitations(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	ownerID, ok := h.getOwnerID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	team, _, err := h.service.GetTeam(c.Request.Context(), ownerID, slug, &userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var query ListInvitationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var status *InvitationStatus
	if query.Status != "" {
		status = &query.Status
	}

	invitations, err := h.service.ListTeamInvitations(c.Request.Context(), team.ID, userID, status, query.Limit, query.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]*InvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = inv.ToResponse(false, "")
	}

	c.JSON(http.StatusOK, gin.H{"invitations": responses})
}

// ListMyInvitations handles listing user's pending invitations.
//
//	@Summary		List my invitations
//	@Description	Get all pending invitations for the current user
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Limit"		default(20)
//	@Param			offset	query		int	false	"Offset"	default(0)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]string
//	@Router			/invitations [get]
func (h *Handler) ListMyInvitations(c *gin.Context) {
	userEmail, ok := h.getUserEmail(c)
	if !ok {
		return
	}

	var query ListInvitationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invitations, err := h.service.ListMyInvitations(c.Request.Context(), userEmail, query.Limit, query.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]*InvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = inv.ToResponse(true, h.baseURL)
	}

	c.JSON(http.StatusOK, gin.H{"invitations": responses})
}

// AcceptInvitation handles accepting an invitation.
//
//	@Summary		Accept invitation
//	@Description	Accept a team invitation
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			token	path		string	true	"Invitation token"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		410		{object}	map[string]string
//	@Router			/invitations/{token}/accept [post]
func (h *Handler) AcceptInvitation(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	userEmail, ok := h.getUserEmail(c)
	if !ok {
		return
	}

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	team, err := h.service.AcceptInvitation(c.Request.Context(), token, userID, userEmail)
	if err != nil {
		h.handleError(c, err)
		return
	}

	memberCount, _ := h.service.GetMemberCount(c.Request.Context(), team.ID)
	member, _ := h.service.repo.GetMember(c.Request.Context(), team.ID, userID)
	var myRole *Role
	if member != nil {
		myRole = &member.Role
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invitation accepted",
		"team":    team.ToResponse(memberCount, myRole),
	})
}

// RejectInvitation handles rejecting an invitation.
//
//	@Summary		Reject invitation
//	@Description	Reject a team invitation
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			token	path		string	true	"Invitation token"
//	@Success		200		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Router			/invitations/{token}/reject [post]
func (h *Handler) RejectInvitation(c *gin.Context) {
	userEmail, ok := h.getUserEmail(c)
	if !ok {
		return
	}

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if err := h.service.RejectInvitation(c.Request.Context(), token, userEmail); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation rejected"})
}

// RevokeInvitation handles revoking an invitation.
//
//	@Summary		Revoke invitation
//	@Description	Revoke a pending invitation
//	@Tags			Collaboration
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Invitation ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/invitations/{id} [delete]
func (h *Handler) RevokeInvitation(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	invitationIDStr := c.Param("id")
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
		return
	}

	if err := h.service.RevokeInvitation(c.Request.Context(), invitationID, userID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked"})
}

// ========== Helper Methods ==========

// getUserID extracts user ID from context.
func (h *Handler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		// Try string
		if idStr, ok := userIDVal.(string); ok {
			var err error
			userID, err = uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
				return uuid.Nil, false
			}
			return userID, true
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, false
	}

	return userID, true
}

// tryGetUserID attempts to get user ID, returns nil if not found.
func (h *Handler) tryGetUserID(c *gin.Context) (*uuid.UUID, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return nil, true
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		if idStr, ok := userIDVal.(string); ok {
			var err error
			userID, err = uuid.Parse(idStr)
			if err != nil {
				return nil, true
			}
			return &userID, true
		}
		return nil, true
	}

	return &userID, true
}

// getUserEmail extracts user email from context.
func (h *Handler) getUserEmail(c *gin.Context) (string, bool) {
	emailVal, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	email, ok := emailVal.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user email"})
		return "", false
	}

	return email, true
}

// getOwnerID extracts owner ID from context or query.
// For now, we use the current user's ID as owner.
func (h *Handler) getOwnerID(c *gin.Context) (uuid.UUID, bool) {
	// First check if owner is specified in query
	ownerIDStr := c.Query("owner_id")
	if ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_id"})
			return uuid.Nil, false
		}
		return ownerID, true
	}

	// Default to current user
	return h.getUserID(c)
}

// handleError handles service errors.
func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTeamNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "team_not_found"})
	case errors.Is(err, ErrSlugAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "slug_already_exists"})
	case errors.Is(err, ErrMemberNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "member_not_found"})
	case errors.Is(err, ErrAlreadyMember):
		c.JSON(http.StatusConflict, gin.H{"error": "user_already_member"})
	case errors.Is(err, ErrMemberLimitExceeded):
		c.JSON(http.StatusForbidden, gin.H{"error": "member_limit_exceeded"})
	case errors.Is(err, ErrCannotChangeOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot_change_owner_role"})
	case errors.Is(err, ErrCannotRemoveOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot_remove_owner"})
	case errors.Is(err, ErrOnlyOwnerCanDelete):
		c.JSON(http.StatusForbidden, gin.H{"error": "only_owner_can_delete"})
	case errors.Is(err, ErrOnlyOwnerCanTransfer):
		c.JSON(http.StatusForbidden, gin.H{"error": "only_owner_can_transfer"})
	case errors.Is(err, ErrInsufficientPermission):
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient_permission"})
	case errors.Is(err, ErrInvalidRole):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_role"})
	case errors.Is(err, ErrInvitationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation_not_found"})
	case errors.Is(err, ErrInvitationExpired):
		c.JSON(http.StatusGone, gin.H{"error": "invitation_expired"})
	case errors.Is(err, ErrInvitationAlreadyProcessed):
		c.JSON(http.StatusGone, gin.H{"error": "invitation_already_processed"})
	case errors.Is(err, ErrInvitationAlreadyPending):
		c.JSON(http.StatusConflict, gin.H{"error": "invitation_already_pending"})
	case errors.Is(err, ErrInvitationNotForYou):
		c.JSON(http.StatusForbidden, gin.H{"error": "invitation_not_for_you"})
	case errors.Is(err, ErrCannotRevokeProcessed):
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot_revoke_processed_invitation"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
