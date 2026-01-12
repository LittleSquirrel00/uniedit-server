package user

import (
	"time"

	commonv1 "github.com/uniedit/server/api/pb/common"
	userv1 "github.com/uniedit/server/api/pb/user"
	"github.com/uniedit/server/internal/model"
)

func toCommonUserPB(u *model.User) *commonv1.User {
	if u == nil {
		return nil
	}

	provider := "email"
	if u.OAuthProvider != nil && *u.OAuthProvider != "" {
		provider = *u.OAuthProvider
	}

	suspendedAt := ""
	if u.SuspendedAt != nil {
		suspendedAt = u.SuspendedAt.UTC().Format(time.RFC3339Nano)
	}

	suspendReason := ""
	if u.SuspendReason != nil {
		suspendReason = *u.SuspendReason
	}

	return &commonv1.User{
		Id:            u.ID.String(),
		Email:         u.Email,
		Name:          u.Name,
		AvatarUrl:     u.AvatarURL,
		Provider:      provider,
		Status:        string(u.Status),
		EmailVerified: u.EmailVerified,
		IsAdmin:       u.IsAdmin,
		SuspendedAt:   suspendedAt,
		SuspendReason: suspendReason,
		CreatedAt:     u.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toProfilePB(p *model.Profile) *userv1.Profile {
	if p == nil {
		return nil
	}
	return &userv1.Profile{
		UserId:      p.UserID.String(),
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarUrl:   p.AvatarURL,
		UpdatedAt:   p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toPreferencesPB(p *model.Preferences) *userv1.Preferences {
	if p == nil {
		return nil
	}
	return &userv1.Preferences{
		UserId:   p.UserID.String(),
		Theme:    p.Theme,
		Language: p.Language,
		Timezone: p.Timezone,
	}
}

