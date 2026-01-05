package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/module/ai/group"
	"github.com/uniedit/server/internal/module/ai/llm"
	"github.com/uniedit/server/internal/module/ai/media"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/task"
)

// Handlers holds all AI handlers.
type Handlers struct {
	Chat  *ChatHandler
	Media *MediaHandler
	Task  *TaskHandler
	Admin *AdminHandler
}

// NewHandlers creates all AI handlers.
func NewHandlers(
	llmService *llm.Service,
	mediaService *media.Service,
	taskManager *task.Manager,
	providerRepo provider.Repository,
	groupRepo group.Repository,
	registry *provider.Registry,
	groupManager *group.Manager,
) *Handlers {
	return &Handlers{
		Chat:  NewChatHandler(llmService),
		Media: NewMediaHandler(mediaService),
		Task:  NewTaskHandler(taskManager),
		Admin: NewAdminHandler(providerRepo, groupRepo, registry, groupManager),
	}
}

// RegisterRoutes registers all AI routes.
func (h *Handlers) RegisterRoutes(r *gin.RouterGroup, adminRouter *gin.RouterGroup) {
	// Public AI routes (require authentication)
	h.Chat.RegisterRoutes(r)
	h.Media.RegisterRoutes(r)
	h.Task.RegisterRoutes(r)

	// Admin routes (require admin role)
	if adminRouter != nil {
		h.Admin.RegisterRoutes(adminRouter)
	}
}

// RegisterPublicRoutes registers only public AI routes.
func (h *Handlers) RegisterPublicRoutes(r *gin.RouterGroup) {
	h.Chat.RegisterRoutes(r)
	h.Media.RegisterRoutes(r)
	h.Task.RegisterRoutes(r)
}

// RegisterAdminRoutes registers only admin AI routes.
func (h *Handlers) RegisterAdminRoutes(r *gin.RouterGroup) {
	h.Admin.RegisterRoutes(r)
}
