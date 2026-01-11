package app

import (
	"github.com/gin-gonic/gin"

	pingv1 "github.com/uniedit/server/api/pb/ping"
	"github.com/uniedit/server/internal/adapter/inbound/http/pingproto"
)

func (a *App) registerProtoRoutes(v1 *gin.RouterGroup) {
	if v1 == nil {
		return
	}

	pingv1.RegisterPingGinServer(v1, pingproto.NewHandler())
}
