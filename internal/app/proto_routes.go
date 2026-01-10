package app

import (
	"github.com/gin-gonic/gin"
	pingv1 "github.com/uniedit/server/api/ping/pb/v1"
)

type pingProtoServer struct{}

func (pingProtoServer) Ping(_ *gin.Context, in *pingv1.PingRequest) (*pingv1.PingReply, error) {
	msg := in.GetMessage()
	if msg == "" {
		msg = "pong"
	}
	return &pingv1.PingReply{Message: msg}, nil
}

func (a *App) registerProtoRoutes(v1 *gin.RouterGroup) {
	pingv1.RegisterPingGinServer(v1, pingProtoServer{})
}
