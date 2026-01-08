package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
)

// userAdminAdapter implements inbound.UserAdminPort.
type userAdminAdapter struct {
	domain user.UserDomain
}

// NewUserAdminAdapter creates a new user admin HTTP adapter.
func NewUserAdminAdapter(domain user.UserDomain) inbound.UserAdminPort {
	return &userAdminAdapter{domain: domain}
}

// RegisterRoutes registers admin user routes.
func (a *userAdminAdapter) RegisterRoutes(r *gin.RouterGroup) {
	admin := r.Group("/admin/users")
	{
		admin.GET("", a.ListUsers)
		admin.GET("/:id", a.GetUser)
		admin.POST("/:id/suspend", a.SuspendUser)
		admin.POST("/:id/reactivate", a.ReactivateUser)
		admin.PUT("/:id/admin", a.SetAdminStatus)
		admin.DELETE("/:id", a.DeleteUser)
	}
}

func (a *userAdminAdapter) ListUsers(c *gin.Context) {
	var filter model.UserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_input",
			Message: err.Error(),
		})
		return
	}

	users, total, err := a.domain.ListUsers(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to response
	responses := make([]*model.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	filter.DefaultPagination()
	c.JSON(http.StatusOK, model.NewPaginatedResponse(responses, total, filter.Page, filter.PageSize))
}

func (a *userAdminAdapter) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	u, err := a.domain.GetUser(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, u.ToResponse())
}

func (a *userAdminAdapter) SuspendUser(c *gin.Context) {
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

	if err := a.domain.SuspendUser(c.Request.Context(), id, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

func (a *userAdminAdapter) ReactivateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	if err := a.domain.ReactivateUser(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user reactivated"})
}

func (a *userAdminAdapter) SetAdminStatus(c *gin.Context) {
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

	if err := a.domain.SetAdminStatus(c.Request.Context(), id, req.IsAdmin); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin status updated"})
}

func (a *userAdminAdapter) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	if err := a.domain.AdminDeleteUser(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// Compile-time check
var _ inbound.UserAdminPort = (*userAdminAdapter)(nil)
