package userproto

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonv1 "github.com/uniedit/server/api/pb/common"
	userv1 "github.com/uniedit/server/api/pb/user"
	"github.com/uniedit/server/internal/domain/user"
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
	out, err := h.userDomain.Register(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}

	c.Status(http.StatusCreated)
	return out, nil
}

func (h *Handler) VerifyEmail(c *gin.Context, in *userv1.VerifyEmailRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.VerifyEmail(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) ResendVerification(c *gin.Context, in *userv1.ResendVerificationRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.ResendVerification(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) RequestPasswordReset(c *gin.Context, in *userv1.RequestPasswordResetRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.RequestPasswordReset(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) CompletePasswordReset(c *gin.Context, in *userv1.CompletePasswordResetRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.ResetPassword(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
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
	return u, nil
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
	return p, nil
}

func (h *Handler) UpdateProfile(c *gin.Context, in *userv1.UpdateProfileRequest) (*commonv1.User, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	u, err := h.userDomain.UpdateProfile(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return u, nil
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
	return prefs, nil
}

func (h *Handler) UpdatePreferences(c *gin.Context, in *userv1.UpdatePreferencesRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.userDomain.UpdatePreferences(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) UploadAvatar(c *gin.Context, in *userv1.UploadAvatarRequest) (*userv1.UploadAvatarResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.userDomain.UploadAvatar(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) ChangePassword(c *gin.Context, in *userv1.ChangePasswordRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.userDomain.ChangePassword(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) DeleteAccount(c *gin.Context, in *userv1.DeleteAccountRequest) (*commonv1.MessageResponse, error) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return nil, &protohttp.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "User not authenticated"}
	}

	out, err := h.userDomain.DeleteAccount(c.Request.Context(), userID, in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) ListUsers(c *gin.Context, in *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	out, err := h.userDomain.ListUsers(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) GetUser(c *gin.Context, in *userv1.GetByIDRequest) (*commonv1.User, error) {
	u, err := h.userDomain.GetUserByID(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return u, nil
}

func (h *Handler) SuspendUser(c *gin.Context, in *userv1.SuspendUserRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.SuspendUser(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) ReactivateUser(c *gin.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.ReactivateUser(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) SetAdminStatus(c *gin.Context, in *userv1.SetAdminStatusRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.SetAdminStatus(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func (h *Handler) DeleteUser(c *gin.Context, in *userv1.GetByIDRequest) (*commonv1.MessageResponse, error) {
	out, err := h.userDomain.AdminDeleteUser(c.Request.Context(), in)
	if err != nil {
		return nil, mapUserError(err)
	}
	return out, nil
}

func mapUserError(err error) error {
	switch {
	case errors.Is(err, user.ErrInvalidRequest):
		return &protohttp.HTTPError{Status: http.StatusBadRequest, Code: "invalid_request", Message: "Invalid request", Err: err}
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
