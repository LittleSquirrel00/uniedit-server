package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SystemRole string

const (
	SystemRoleAdmin SystemRole = "admin"
	SystemRoleSRE   SystemRole = "sre"
)

type SystemRoleAuthorizer struct {
	adminEmails map[string]struct{}
	sreEmails   map[string]struct{}

	adminUserIDs map[uuid.UUID]struct{}
	sreUserIDs   map[uuid.UUID]struct{}

	userAdminChecker func(ctx context.Context, userID uuid.UUID) (bool, error)
}

type SystemRoleAuthorizerOption func(*SystemRoleAuthorizer)

// WithUserAdminChecker enables DB-backed admin flag checks (e.g. user.is_admin).
func WithUserAdminChecker(fn func(ctx context.Context, userID uuid.UUID) (bool, error)) SystemRoleAuthorizerOption {
	return func(a *SystemRoleAuthorizer) {
		a.userAdminChecker = fn
	}
}

func NewSystemRoleAuthorizer(adminEmails, sreEmails, adminUserIDs, sreUserIDs []string, opts ...SystemRoleAuthorizerOption) *SystemRoleAuthorizer {
	a := &SystemRoleAuthorizer{
		adminEmails:  normalizeEmailSet(adminEmails),
		sreEmails:    normalizeEmailSet(sreEmails),
		adminUserIDs: parseUUIDSet(adminUserIDs),
		sreUserIDs:   parseUUIDSet(sreUserIDs),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	return a
}

func (a *SystemRoleAuthorizer) HasAny(ctx context.Context, userID uuid.UUID, email string, roles ...SystemRole) (bool, error) {
	if userID == uuid.Nil && email == "" {
		return false, nil
	}

	email = normalizeEmail(email)

	hasRole := func(role SystemRole) (bool, error) {
		switch role {
		case SystemRoleAdmin:
			if email != "" {
				if _, ok := a.adminEmails[email]; ok {
					return true, nil
				}
			}
			if userID != uuid.Nil {
				if _, ok := a.adminUserIDs[userID]; ok {
					return true, nil
				}
				if a.userAdminChecker != nil {
					ok, err := a.userAdminChecker(ctx, userID)
					if err != nil {
						return false, err
					}
					if ok {
						return true, nil
					}
				}
			}
			return false, nil
		case SystemRoleSRE:
			if email != "" {
				if _, ok := a.sreEmails[email]; ok {
					return true, nil
				}
			}
			if userID != uuid.Nil {
				if _, ok := a.sreUserIDs[userID]; ok {
					return true, nil
				}
			}
			return false, nil
		default:
			return false, nil
		}
	}

	for _, role := range roles {
		ok, err := hasRole(role)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

func RequireAnySystemRole(authorizer *SystemRoleAuthorizer, roles ...SystemRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User not authenticated",
				},
			})
			return
		}

		if authorizer == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
			return
		}

		email := GetEmail(c)
		allowed, err := authorizer.HasAny(c.Request.Context(), userID, email, roles...)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Permission check failed",
				},
			})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
			return
		}

		c.Next()
	}
}

func RequireAdmin(authorizer *SystemRoleAuthorizer) gin.HandlerFunc {
	return RequireAnySystemRole(authorizer, SystemRoleAdmin)
}

func RequireSRE(authorizer *SystemRoleAuthorizer) gin.HandlerFunc {
	return RequireAnySystemRole(authorizer, SystemRoleSRE)
}

func RequireAdminOrSRE(authorizer *SystemRoleAuthorizer) gin.HandlerFunc {
	return RequireAnySystemRole(authorizer, SystemRoleAdmin, SystemRoleSRE)
}

func normalizeEmailSet(emails []string) map[string]struct{} {
	out := make(map[string]struct{}, len(emails))
	for _, e := range emails {
		e = normalizeEmail(e)
		if e == "" {
			continue
		}
		out[e] = struct{}{}
	}
	return out
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func parseUUIDSet(ids []string) map[uuid.UUID]struct{} {
	out := make(map[uuid.UUID]struct{}, len(ids))
	for _, s := range ids {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}
