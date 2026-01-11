package userproto

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonv1 "github.com/uniedit/server/api/pb/common"
	userv1 "github.com/uniedit/server/api/pb/user"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/transport/protohttp"
	"github.com/uniedit/server/internal/utils/middleware"
)

type Handler struct {
	userDomain user.UserDomain
}

func NewHandler(userDomain user.UserDomain) *Handler {
	return &Handler{userDomain: userDomain}
}

func (h *Handler) Register(c *gin.Context, in *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	u, err := h.userDomain.Register(c.Request.Context(), &user.RegisterInput{
		Email:    in.GetEmail(),
		Password: in.GetPassword(),
		Name:     in.GetName(),
	})
	if err != nil {
		return nil, mapUserError(err)
	}

	c.Status(http.StatusCreated)
	return &userv1.RegisterResponse{
		Message: "registration successful, please check your email for verification",
		User:    toCommonUser(u),
	}, nil
}

func (h *Handler) VerifyEmail(c *gin.Context, in *userv1.VerifyEmailRequest) (*commonv1.MessageResponse, error) {
	if err := h.userDomain.VerifyEmail(c.Request.Context(), in.GetToken()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "email verified successfully"}, nil
}

func (h *Handler) ResendVerification(c *gin.Context, in *userv1.ResendVerificationRequest) (*commonv1.MessageResponse, error) {
	if err := h.userDomain.ResendVerification(c.Request.Context(), in.GetEmail()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "verification email sent if account exists"}, nil
}

func (h *Handler) RequestPasswordReset(c *gin.Context, in *userv1.RequestPasswordResetRequest) (*commonv1.MessageResponse, error) {
	if err := h.userDomain.RequestPasswordReset(c.Request.Context(), in.GetEmail()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "password reset email sent if account exists"}, nil
}

func (h *Handler) CompletePasswordReset(c *gin.Context, in *userv1.CompletePasswordResetRequest) (*commonv1.MessageResponse, error) {
	if err := h.userDomain.ResetPassword(c.Request.Context(), in.GetToken(), in.GetNewPassword()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "password reset successfully"}, nil
}

func (h *Handler) GetMe(c *gin.Context, _ *commonv1.Empty) (*commonv1.User, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	u, err := h.userDomain.GetUser(c.Request.Context(), userID)
	if err != nil {
		return nil, mapUserError(err)
	}
	return toCommonUser(u), nil
}

func (h *Handler) GetProfile(c *gin.Context, _ *commonv1.Empty) (*userv1.Profile, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	p, err := h.userDomain.GetProfile(c.Request.Context(), userID)
	if err != nil {
		return nil, mapUserError(err)
	}
	return &userv1.Profile{
		UserId:      p.UserID.String(),
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarUrl:   p.AvatarURL,
		UpdatedAt:   p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (h *Handler) UpdateProfile(c *gin.Context, in *userv1.UpdateProfileRequest) (*commonv1.User, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	var name *string
	if in.GetName() != "" {
		v := in.GetName()
		name = &v
	}
	var avatarURL *string
	if in.GetAvatarUrl() != "" {
		v := in.GetAvatarUrl()
		avatarURL = &v
	}

	u, err := h.userDomain.UpdateProfile(c.Request.Context(), userID, &user.UpdateProfileInput{
		Name:      name,
		AvatarURL: avatarURL,
	})
	if err != nil {
		return nil, mapUserError(err)
	}
	return toCommonUser(u), nil
}

func (h *Handler) GetPreferences(c *gin.Context, _ *commonv1.Empty) (*userv1.Preferences, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	prefs, err := h.userDomain.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		return nil, mapUserError(err)
	}
	return &userv1.Preferences{
		UserId:   prefs.UserID.String(),
		Theme:    prefs.Theme,
		Language: prefs.Language,
		Timezone: prefs.Timezone,
	}, nil
}

func (h *Handler) UpdatePreferences(c *gin.Context, in *userv1.UpdatePreferencesRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	if err := h.userDomain.UpdatePreferences(c.Request.Context(), userID, &model.Preferences{
		Theme:    in.GetTheme(),
		Language: in.GetLanguage(),
		Timezone: in.GetTimezone(),
	}); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "preferences updated"}, nil
}

func (h *Handler) UploadAvatar(c *gin.Context, in *userv1.UploadAvatarRequest) (*userv1.UploadAvatarResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	url, err := h.userDomain.UploadAvatar(c.Request.Context(), userID, in.GetData(), in.GetContentType())
	if err != nil {
		return nil, mapUserError(err)
	}
	return &userv1.UploadAvatarResponse{Url: url}, nil
}

func (h *Handler) ChangePassword(c *gin.Context, in *userv1.ChangePasswordRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	if err := h.userDomain.ChangePassword(c.Request.Context(), userID, in.GetCurrentPassword(), in.GetNewPassword()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "password changed"}, nil
}

func (h *Handler) DeleteAccount(c *gin.Context, in *userv1.DeleteAccountRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	if err := h.userDomain.DeleteAccount(c.Request.Context(), userID, in.GetPassword()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "account deleted"}, nil
}

func (h *Handler) ListUsers(c *gin.Context, in *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	filter := model.UserFilter{}

	if len(in.GetIds()) > 0 {
		filter.IDs = make([]uuid.UUID, 0, len(in.GetIds()))
		for _, idStr := range in.GetIds() {
			if idStr == "" {
				continue
			}
			id, err := uuid.Parse(idStr)
			if err != nil {
				return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid user ID", Err: err}
			}
			filter.IDs = append(filter.IDs, id)
		}
	}
	if in.GetEmail() != "" {
		filter.Email = in.GetEmail()
	}
	if in.GetStatus() != "" {
		status := model.UserStatus(in.GetStatus())
		if !status.IsValid() {
			return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_status", Message: "Invalid user status"}
		}
		filter.Status = &status
	}
	if in.GetSearch() != "" {
		filter.Search = in.GetSearch()
	}
	if in.GetPage() > 0 {
		filter.Page = int(in.GetPage())
	}
	if in.GetPageSize() > 0 {
		filter.PageSize = int(in.GetPageSize())
	}
	filter.DefaultPagination()

	if in.GetIsAdmin() != nil {
		v := in.GetIsAdmin().GetValue()
		filter.IsAdmin = &v
	}

	users, total, err := h.userDomain.ListUsers(c.Request.Context(), filter)
	if err != nil {
		return nil, mapUserError(err)
	}

	out := make([]*commonv1.User, 0, len(users))
	for _, u := range users {
		out = append(out, toCommonUser(u))
	}

	totalPages := int32(0)
	if filter.PageSize > 0 {
		totalPages = int32((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}

	return &userv1.ListUsersResponse{
		Data:       out,
		Total:      total,
		Page:       int32(filter.Page),
		PageSize:   int32(filter.PageSize),
		TotalPages: totalPages,
	}, nil
}

func (h *Handler) GetUser(c *gin.Context, in *userv1.GetByIDRequest) (*commonv1.User, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid user ID", Err: err}
	}

	u, err := h.userDomain.GetUser(c.Request.Context(), id)
	if err != nil {
		return nil, mapUserError(err)
	}
	return toCommonUser(u), nil
}

func (h *Handler) SuspendUser(c *gin.Context, in *userv1.SuspendUserRequest) (*commonv1.MessageResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid user ID", Err: err}
	}

	if err := h.userDomain.SuspendUser(c.Request.Context(), id, in.GetReason()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "user suspended"}, nil
}

func (h *Handler) ReactivateUser(c *gin.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid user ID", Err: err}
	}

	if err := h.userDomain.ReactivateUser(c.Request.Context(), id); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "user reactivated"}, nil
}

func (h *Handler) SetAdminStatus(c *gin.Context, in *userv1.SetAdminStatusRequest) (*commonv1.MessageResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid user ID", Err: err}
	}

	if err := h.userDomain.SetAdminStatus(c.Request.Context(), id, in.GetIsAdmin()); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "admin status updated"}, nil
}

func (h *Handler) DeleteUser(c *gin.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	id, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_id", Message: "Invalid user ID", Err: err}
	}

	if err := h.userDomain.AdminDeleteUser(c.Request.Context(), id); err != nil {
		return nil, mapUserError(err)
	}
	return &commonv1.MessageResponse{Message: "user deleted"}, nil
}

func mapUserError(err error) error {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		return &protohttp.HTTPError{Status: http.StatusNotFound, Code: "user_not_found", Message: "User not found", Err: err}
	case errors.Is(err, user.ErrEmailAlreadyExists):
		return &protohttp.HTTPError{Status: http.StatusConflict, Code: "email_exists", Message: "Email already registered", Err: err}
	case errors.Is(err, user.ErrInvalidCredentials):
		return &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "invalid_credentials", Message: "Invalid credentials", Err: err}
	case errors.Is(err, user.ErrEmailNotVerified):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "email_not_verified", Message: "Email not verified", Err: err}
	case errors.Is(err, user.ErrAccountSuspended):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "account_suspended", Message: "Account suspended", Err: err}
	case errors.Is(err, user.ErrAccountDeleted):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "account_deleted", Message: "Account deleted", Err: err}
	case errors.Is(err, user.ErrIncorrectPassword):
		return &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "incorrect_password", Message: "Incorrect password", Err: err}
	case errors.Is(err, user.ErrForbidden):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: "Forbidden", Err: err}
	case errors.Is(err, user.ErrPasswordTooShort):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "password_too_short", Message: "Password must be at least 8 characters", Err: err}
	case errors.Is(err, user.ErrPasswordRequired):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "password_required", Message: "Password required for email users", Err: err}
	case errors.Is(err, user.ErrInvalidToken):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_token", Message: "Invalid verification token", Err: err}
	case errors.Is(err, user.ErrTokenExpired):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "token_expired", Message: "Verification token expired", Err: err}
	case errors.Is(err, user.ErrTokenAlreadyUsed):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "token_used", Message: "Verification token already used", Err: err}
	case errors.Is(err, user.ErrCannotSuspendAdmin):
		return &protohttp.HTTPError{Status: http.StatusForbidden, Code: "cannot_suspend_admin", Message: "Cannot suspend admin user", Err: err}
	case errors.Is(err, user.ErrUserAlreadyActive):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "user_already_active", Message: "User is already active", Err: err}
	default:
		return err
	}
}

func toCommonUser(u *model.User) *commonv1.User {
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
