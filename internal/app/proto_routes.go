package app

import (
	"github.com/gin-gonic/gin"

	authv1 "github.com/uniedit/server/api/pb/auth"
	orderv1 "github.com/uniedit/server/api/pb/order"
	pingv1 "github.com/uniedit/server/api/pb/ping"
	userv1 "github.com/uniedit/server/api/pb/user"
)

func (a *App) registerProtoRoutes(v1 *gin.RouterGroup, protected *gin.RouterGroup, admin *gin.RouterGroup) {
	if v1 == nil {
		return
	}

	pingv1.RegisterPingGinServer(v1, a.pingProtoHandler)

	if protected != nil {
		authv1.RegisterAuthSessionServiceGinServer(protected, a.authProtoHandler)
		authv1.RegisterAuthAPIKeyServiceGinServer(protected, a.authProtoHandler)

		orderv1.RegisterOrderServiceGinServer(protected, a.orderProtoHandler)

		userv1.RegisterUserProfileServiceGinServer(protected, a.userProtoHandler)
		if admin != nil {
			userv1.RegisterUserAdminServiceGinServer(admin, a.userProtoHandler)
		}
	}

	authv1.RegisterAuthOAuthServiceGinServer(v1, a.authProtoHandler)
	userv1.RegisterUserAuthServiceGinServer(v1, a.userProtoHandler)

	if admin != nil {
		authv1.RegisterAuthSystemAPIKeyServiceGinServer(admin, a.authProtoHandler)
	}
}
