package pingproto

import (
	"strings"

	"github.com/gin-gonic/gin"

	pingv1 "github.com/uniedit/server/api/pb/ping"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Ping(c *gin.Context, in *pingv1.PingRequest) (*pingv1.PingReply, error) {
	_ = h

	msg := ""
	if in != nil {
		msg = strings.TrimSpace(in.Message)
	}
	if msg == "" {
		msg = "pong"
	}

	return &pingv1.PingReply{Message: msg}, nil
}
