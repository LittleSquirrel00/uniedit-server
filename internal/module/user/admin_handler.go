package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler handles admin HTTP requests for user management.
type AdminHandler struct {
	service *Service
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(service *Service) *AdminHandler {
	return &AdminHandler{service: service}
}

// RegisterRoutes registers the admin routes.
func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
	admin := r.Group("/admin/users")
	{
		admin.GET("", h.ListUsers)
		admin.GET("/:id", h.GetUser)
		admin.POST("/:id/suspend", h.SuspendUser)
		admin.POST("/:id/reactivate", h.ReactivateUser)
		admin.PUT("/:id/admin-status", h.SetAdminStatus)
	}
}

// ListUsers returns a paginated list of users.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var filter UserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pagination := NewPagination()
	if err := c.ShouldBindQuery(pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	users, total, err := h.service.ListUsers(c.Request.Context(), &filter, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	// Convert to responses
	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, UserListResponse{
		Users:      responses,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	})
}

// GetUser returns a user by ID.
func (h *AdminHandler) GetUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// SuspendUser suspends a user account.
func (h *AdminHandler) SuspendUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req SuspendUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SuspendUser(c.Request.Context(), userID, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	// Return updated user
	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, MessageResponse{Message: "User suspended"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// ReactivateUser reactivates a suspended user.
func (h *AdminHandler) ReactivateUser(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.ReactivateUser(c.Request.Context(), userID); err != nil {
		handleError(c, err)
		return
	}

	// Return updated user
	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, MessageResponse{Message: "User reactivated"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// SetAdminStatus sets a user's admin status.
func (h *AdminHandler) SetAdminStatus(c *gin.Context) {
	if !isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req SetAdminStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SetAdminStatus(c.Request.Context(), userID, req.IsAdmin); err != nil {
		handleError(c, err)
		return
	}

	// Return updated user
	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, MessageResponse{Message: "Admin status updated"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// --- Helpers ---

func isAdmin(c *gin.Context) bool {
	isAdminVal, exists := c.Get("is_admin")
	if !exists {
		return false
	}
	isAdmin, ok := isAdminVal.(bool)
	return ok && isAdmin
}
