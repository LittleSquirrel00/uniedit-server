package app

import (
	"github.com/gin-gonic/gin"

	authv1 "github.com/uniedit/server/api/pb/auth"
	aiv1 "github.com/uniedit/server/api/pb/ai"
	billingv1 "github.com/uniedit/server/api/pb/billing"
	collabv1 "github.com/uniedit/server/api/pb/collaboration"
	gitv1 "github.com/uniedit/server/api/pb/git"
	mediav1 "github.com/uniedit/server/api/pb/media"
	orderv1 "github.com/uniedit/server/api/pb/order"
	paymentv1 "github.com/uniedit/server/api/pb/payment"
	pingv1 "github.com/uniedit/server/api/pb/ping"
	userv1 "github.com/uniedit/server/api/pb/user"
)

func (a *App) registerProtoRoutes(v1 *gin.RouterGroup, protected *gin.RouterGroup, admin *gin.RouterGroup) {
	if v1 == nil {
		return
	}

	pingv1.RegisterPingGinServer(v1, a.pingProtoHandler)

	if a.billingProtoHandler != nil {
		billingv1.RegisterBillingPublicServiceGinServer(v1, a.billingProtoHandler)
	}

	if a.paymentProtoHandler != nil {
		paymentv1.RegisterWebhookServiceGinServer(v1, a.paymentProtoHandler)
	}

	if a.gitProtoHandler != nil {
		gitv1.RegisterGitPublicServiceGinServer(v1, a.gitProtoHandler)
	}

	if protected != nil {
		authv1.RegisterAuthSessionServiceGinServer(protected, a.authProtoHandler)
		authv1.RegisterAuthAPIKeyServiceGinServer(protected, a.authProtoHandler)

		orderv1.RegisterOrderServiceGinServer(protected, a.orderProtoHandler)

		userv1.RegisterUserProfileServiceGinServer(protected, a.userProtoHandler)
		if admin != nil {
			userv1.RegisterUserAdminServiceGinServer(admin, a.userProtoHandler)
		}

		if a.billingProtoHandler != nil {
			billingv1.RegisterBillingServiceGinServer(protected, a.billingProtoHandler)
		}
		if a.aiProtoHandler != nil {
			aiv1.RegisterAIServiceGinServer(protected, a.aiProtoHandler)
		}
		if a.collaborationProtoHandler != nil {
			collabv1.RegisterCollaborationServiceGinServer(protected, a.collaborationProtoHandler)
		}
		if a.paymentProtoHandler != nil {
			paymentv1.RegisterPaymentServiceGinServer(protected, a.paymentProtoHandler)
		}
		if a.gitProtoHandler != nil {
			gitv1.RegisterGitServiceGinServer(protected, a.gitProtoHandler)
		}
		if a.mediaProtoHandler != nil {
			mediav1.RegisterMediaServiceGinServer(protected, a.mediaProtoHandler)
		}
	}

	authv1.RegisterAuthOAuthServiceGinServer(v1, a.authProtoHandler)
	userv1.RegisterUserAuthServiceGinServer(v1, a.userProtoHandler)

	if admin != nil {
		authv1.RegisterAuthSystemAPIKeyServiceGinServer(admin, a.authProtoHandler)
		if a.billingProtoHandler != nil {
			billingv1.RegisterBillingAdminServiceGinServer(admin, a.billingProtoHandler)
		}
		if a.aiProtoHandler != nil {
			aiv1.RegisterAIAdminServiceGinServer(admin, a.aiProtoHandler)
		}
		if a.paymentProtoHandler != nil {
			paymentv1.RegisterPaymentAdminServiceGinServer(admin, a.paymentProtoHandler)
		}
		if a.mediaProtoHandler != nil {
			mediav1.RegisterMediaAdminServiceGinServer(admin, a.mediaProtoHandler)
		}
	}
}
