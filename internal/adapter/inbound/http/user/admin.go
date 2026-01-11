package userhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// AdminHandler handles user admin HTTP requests.
type AdminHandler struct {
	domain user.UserDomain
}

// NewAdminHandler creates a new user admin handler.
func NewAdminHandler(domain user.UserDomain) *AdminHandler {
	return &AdminHandler{domain: domain}
}

// RegisterRoutes registers admin user routes.
func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
	admin := r.Group("/users")
	{
		admin.GET("", h.ListUsers)
		admin.GET("/:id", h.GetUser)
		admin.POST("/:id/suspend", h.SuspendUser)
		admin.POST("/:id/reactivate", h.ReactivateUser)
		admin.PUT("/:id/admin", h.SetAdminStatus)
		admin.DELETE("/:id", h.DeleteUser)
	}
}

// ListUsers handles GET /admin/users.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var filter model.UserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	users, total, err := h.domain.ListUsers(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}

	responses := make([]*model.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	filter.DefaultPagination()
	c.JSON(http.StatusOK, model.NewPaginatedResponse(responses, total, filter.Page, filter.PageSize))
}

// GetUser handles GET /admin/users/:id.
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	u, err := h.domain.GetUser(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, u.ToResponse())
}

// SuspendUser handles POST /admin/users/:id/suspend.
func (h *AdminHandler) SuspendUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := h.domain.SuspendUser(c.Request.Context(), id, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

// ReactivateUser handles POST /admin/users/:id/reactivate.
func (h *AdminHandler) ReactivateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	if err := h.domain.ReactivateUser(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user reactivated"})
}

// SetAdminStatus handles PUT /admin/users/:id/admin.
func (h *AdminHandler) SetAdminStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	var req struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	if err := h.domain.SetAdminStatus(c.Request.Context(), id, req.IsAdmin); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin status updated"})
}

// DeleteUser handles DELETE /admin/users/:id.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	if err := h.domain.AdminDeleteUser(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// Compile-time check
var _ inbound.UserAdminPort = (*AdminHandler)(nil)
