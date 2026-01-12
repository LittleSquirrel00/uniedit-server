package collaboration

import (
	"time"

	collabv1 "github.com/uniedit/server/api/pb/collaboration"
	"github.com/uniedit/server/internal/model"
)

func toTeamPB(team *model.Team, memberCount int, myRole *model.TeamRole) *collabv1.Team {
	if team == nil {
		return nil
	}

	role := ""
	if myRole != nil {
		role = string(*myRole)
	}

	return &collabv1.Team{
		Id:          team.ID.String(),
		OwnerId:     team.OwnerID.String(),
		Name:        team.Name,
		Slug:        team.Slug,
		Description: team.Description,
		Visibility:  string(team.Visibility),
		MemberCount: int32(memberCount),
		MemberLimit: int32(team.MemberLimit),
		CreatedAt:   formatTime(team.CreatedAt),
		UpdatedAt:   formatTime(team.UpdatedAt),
		MyRole:      role,
	}
}

func toMemberPB(m *model.TeamMemberWithUser) *collabv1.Member {
	if m == nil {
		return nil
	}
	return &collabv1.Member{
		UserId:   m.UserID.String(),
		Email:    m.Email,
		Name:     m.Name,
		Role:     string(m.Role),
		JoinedAt: formatTime(m.JoinedAt),
	}
}

func toInvitationPB(inv *model.TeamInvitation, includeToken bool, baseURL string) *collabv1.Invitation {
	if inv == nil {
		return nil
	}

	acceptedAt := ""
	if inv.AcceptedAt != nil {
		acceptedAt = formatTime(*inv.AcceptedAt)
	}

	out := &collabv1.Invitation{
		Id:           inv.ID.String(),
		TeamId:       inv.TeamID.String(),
		InviterId:    inv.InviterID.String(),
		InviteeEmail: inv.InviteeEmail,
		Role:         string(inv.Role),
		Status:       string(inv.Status),
		ExpiresAt:    formatTime(inv.ExpiresAt),
		CreatedAt:    formatTime(inv.CreatedAt),
		AcceptedAt:   acceptedAt,
	}

	if inv.Team != nil {
		out.TeamName = inv.Team.Name
	}
	if inv.Inviter != nil {
		out.InviterName = inv.Inviter.Name
	}

	if includeToken {
		out.Token = inv.Token
		if baseURL != "" {
			out.AcceptUrl = baseURL + "/invitations/" + inv.Token + "/accept"
		}
	}

	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

