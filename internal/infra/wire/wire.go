//go:build wireinject

package wire

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	// Domain
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/collaboration"
	domainGit "github.com/uniedit/server/internal/domain/git"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"

	// Infrastructure
	"github.com/uniedit/server/internal/infra/persistence"

	// Application - Commands
	authCmd "github.com/uniedit/server/internal/app/command/auth"
	billingCmd "github.com/uniedit/server/internal/app/command/billing"
	collabCmd "github.com/uniedit/server/internal/app/command/collaboration"
	orderCmd "github.com/uniedit/server/internal/app/command/order"
	paymentCmd "github.com/uniedit/server/internal/app/command/payment"
	userCmd "github.com/uniedit/server/internal/app/command/user"

	// Application - Queries
	authQuery "github.com/uniedit/server/internal/app/query/auth"
	billingQuery "github.com/uniedit/server/internal/app/query/billing"
	collabQuery "github.com/uniedit/server/internal/app/query/collaboration"
	orderQuery "github.com/uniedit/server/internal/app/query/order"
	paymentQuery "github.com/uniedit/server/internal/app/query/payment"
	userQuery "github.com/uniedit/server/internal/app/query/user"

	// Ports - HTTP
	portshttp "github.com/uniedit/server/internal/ports/http"
)

// RepositorySet provides all repository implementations.
var RepositorySet = wire.NewSet(
	persistence.NewBillingRepository,
	wire.Bind(new(billing.Repository), new(*persistence.BillingRepository)),

	persistence.NewOrderRepository,
	wire.Bind(new(order.Repository), new(*persistence.OrderRepository)),

	persistence.NewPaymentRepository,
	wire.Bind(new(payment.Repository), new(*persistence.PaymentRepository)),

	persistence.NewUserRepository,
	wire.Bind(new(user.Repository), new(*persistence.UserRepository)),

	persistence.NewRefreshTokenRepository,
	wire.Bind(new(auth.RefreshTokenRepository), new(*persistence.RefreshTokenRepository)),

	persistence.NewUserAPIKeyRepository,
	wire.Bind(new(auth.UserAPIKeyRepository), new(*persistence.UserAPIKeyRepository)),

	persistence.NewSystemAPIKeyRepository,
	wire.Bind(new(auth.SystemAPIKeyRepository), new(*persistence.SystemAPIKeyRepository)),

	// Collaboration repositories
	persistence.NewCollaborationUnitOfWork,
	wire.Bind(new(collaboration.UnitOfWork), new(*persistence.CollaborationUnitOfWork)),

	persistence.NewTeamRepository,
	wire.Bind(new(collaboration.TeamRepository), new(*persistence.TeamRepository)),

	persistence.NewMemberRepository,
	wire.Bind(new(collaboration.MemberRepository), new(*persistence.MemberRepository)),

	persistence.NewInvitationRepository,
	wire.Bind(new(collaboration.InvitationRepository), new(*persistence.InvitationRepository)),

	persistence.NewUserLookupAdapter,
	wire.Bind(new(collaboration.UserLookup), new(*persistence.UserLookupAdapter)),

	// Git repositories
	persistence.NewGitRepositoryRepository,
	wire.Bind(new(domainGit.RepositoryRepository), new(*persistence.GitRepositoryRepository)),

	persistence.NewGitCollaboratorRepository,
	wire.Bind(new(domainGit.CollaboratorRepository), new(*persistence.GitCollaboratorRepository)),

	persistence.NewGitPullRequestRepository,
	wire.Bind(new(domainGit.PullRequestRepository), new(*persistence.GitPullRequestRepository)),

	persistence.NewGitLFSRepository,
	wire.Bind(new(domainGit.LFSRepository), new(*persistence.GitLFSRepository)),
)

// BillingCommandSet provides billing command handlers.
var BillingCommandSet = wire.NewSet(
	billingCmd.NewCreateSubscriptionHandler,
	billingCmd.NewCancelSubscriptionHandler,
)

// BillingQuerySet provides billing query handlers.
var BillingQuerySet = wire.NewSet(
	billingQuery.NewGetSubscriptionHandler,
	billingQuery.NewListPlansHandler,
	billingQuery.NewGetUsageStatsHandler,
)

// OrderCommandSet provides order command handlers.
var OrderCommandSet = wire.NewSet(
	orderCmd.NewCreateOrderHandler,
	orderCmd.NewCancelOrderHandler,
)

// OrderQuerySet provides order query handlers.
var OrderQuerySet = wire.NewSet(
	orderQuery.NewGetOrderHandler,
	orderQuery.NewListOrdersHandler,
)

// PaymentCommandSet provides payment command handlers.
var PaymentCommandSet = wire.NewSet(
	paymentCmd.NewCreatePaymentHandler,
	paymentCmd.NewMarkPaymentSucceededHandler,
	paymentCmd.NewRefundPaymentHandler,
)

// PaymentQuerySet provides payment query handlers.
var PaymentQuerySet = wire.NewSet(
	paymentQuery.NewGetPaymentHandler,
	paymentQuery.NewListPaymentsByOrderHandler,
)

// UserCommandSet provides user command handlers.
var UserCommandSet = wire.NewSet(
	userCmd.NewRegisterHandler,
	userCmd.NewVerifyEmailHandler,
	userCmd.NewResendVerificationHandler,
	userCmd.NewRequestPasswordResetHandler,
	userCmd.NewResetPasswordHandler,
	userCmd.NewChangePasswordHandler,
	userCmd.NewUpdateProfileHandler,
	userCmd.NewDeleteAccountHandler,
	userCmd.NewSuspendUserHandler,
	userCmd.NewReactivateUserHandler,
	userCmd.NewSetAdminStatusHandler,
)

// UserQuerySet provides user query handlers.
var UserQuerySet = wire.NewSet(
	userQuery.NewGetUserHandler,
	userQuery.NewGetUserByEmailHandler,
	userQuery.NewListUsersHandler,
)

// AuthQuerySet provides auth query handlers.
var AuthQuerySet = wire.NewSet(
	authQuery.NewListUserAPIKeysHandler,
	authQuery.NewListSystemAPIKeysHandler,
	authQuery.NewGetSystemAPIKeyHandler,
)

// CollaborationCommandSet provides collaboration command handlers.
var CollaborationCommandSet = wire.NewSet(
	collabCmd.NewCreateTeamHandler,
	collabCmd.NewUpdateTeamHandler,
	collabCmd.NewDeleteTeamHandler,
	collabCmd.NewSendInvitationHandler,
	collabCmd.NewAcceptInvitationHandler,
	collabCmd.NewRejectInvitationHandler,
	collabCmd.NewRevokeInvitationHandler,
	collabCmd.NewUpdateMemberRoleHandler,
	collabCmd.NewRemoveMemberHandler,
	collabCmd.NewLeaveTeamHandler,
)

// CollaborationQuerySet provides collaboration query handlers.
var CollaborationQuerySet = wire.NewSet(
	collabQuery.NewGetTeamHandler,
	collabQuery.NewListMyTeamsHandler,
	collabQuery.NewListMembersHandler,
	collabQuery.NewListInvitationsHandler,
	collabQuery.NewListMyInvitationsHandler,
)

// HTTPHandlerSet provides HTTP handlers.
var HTTPHandlerSet = wire.NewSet(
	portshttp.NewBillingHandler,
	portshttp.NewOrderHandler,
	portshttp.NewPaymentHandler,
	portshttp.NewUserHandler,
	// Note: CollaborationHandler requires baseURL configuration and should be
	// created separately in the application layer.
)

// HSTSet combines all HST layer providers.
var HSTSet = wire.NewSet(
	RepositorySet,
	BillingCommandSet,
	BillingQuerySet,
	OrderCommandSet,
	OrderQuerySet,
	PaymentCommandSet,
	PaymentQuerySet,
	UserCommandSet,
	UserQuerySet,
	AuthQuerySet,
	CollaborationCommandSet,
	CollaborationQuerySet,
	HTTPHandlerSet,
)

// BillingHandlers contains all billing command and query handlers.
type BillingHandlers struct {
	CreateSubscription *billingCmd.CreateSubscriptionHandler
	CancelSubscription *billingCmd.CancelSubscriptionHandler
	GetSubscription    *billingQuery.GetSubscriptionHandler
	ListPlans          *billingQuery.ListPlansHandler
	GetUsageStats      *billingQuery.GetUsageStatsHandler
}

// OrderHandlers contains all order command and query handlers.
type OrderHandlers struct {
	CreateOrder  *orderCmd.CreateOrderHandler
	CancelOrder  *orderCmd.CancelOrderHandler
	GetOrder     *orderQuery.GetOrderHandler
	ListOrders   *orderQuery.ListOrdersHandler
}

// PaymentHandlers contains all payment command and query handlers.
type PaymentHandlers struct {
	CreatePayment       *paymentCmd.CreatePaymentHandler
	MarkSucceeded       *paymentCmd.MarkPaymentSucceededHandler
	RefundPayment       *paymentCmd.RefundPaymentHandler
	GetPayment          *paymentQuery.GetPaymentHandler
	ListPaymentsByOrder *paymentQuery.ListPaymentsByOrderHandler
}

// HSTHandlers contains all HTTP handlers for the HST architecture.
type HSTHandlers struct {
	Billing *portshttp.BillingHandler
	Order   *portshttp.OrderHandler
	Payment *portshttp.PaymentHandler
	User    *portshttp.UserHandler
	// Note: AuthHandler and CollaborationHandler require external dependencies
	// (OAuthRegistry, JWTGenerator, baseURL, etc.) and should be created
	// separately in the application layer.
}

// AuthHandlers contains all auth command and query handlers.
// Note: AuthHandler requires external dependencies that must be provided separately:
// - OAuthRegistry: OAuth provider registry
// - OAuthStateStore: OAuth state storage (typically Redis)
// - JWTGenerator: JWT token generator
// - CryptoManager: Encryption/decryption manager
// - TokenValidator: Access token validator
type AuthHandlers struct {
	// Token Commands
	RefreshTokens *authCmd.RefreshTokensHandler
	Logout        *authCmd.LogoutHandler

	// User API Key Commands
	CreateUserAPIKey  *authCmd.CreateUserAPIKeyHandler
	DeleteUserAPIKey  *authCmd.DeleteUserAPIKeyHandler
	RotateUserAPIKey  *authCmd.RotateUserAPIKeyHandler

	// System API Key Commands
	CreateSystemAPIKey   *authCmd.CreateSystemAPIKeyHandler
	UpdateSystemAPIKey   *authCmd.UpdateSystemAPIKeyHandler
	DeleteSystemAPIKey   *authCmd.DeleteSystemAPIKeyHandler
	RotateSystemAPIKey   *authCmd.RotateSystemAPIKeyHandler
	ValidateSystemAPIKey *authCmd.ValidateSystemAPIKeyHandler

	// Queries
	ListUserAPIKeys   *authQuery.ListUserAPIKeysHandler
	ListSystemAPIKeys *authQuery.ListSystemAPIKeysHandler
	GetSystemAPIKey   *authQuery.GetSystemAPIKeyHandler
}

// CollaborationHandlers contains all collaboration command and query handlers.
type CollaborationHandlers struct {
	// Team Commands
	CreateTeam *collabCmd.CreateTeamHandler
	UpdateTeam *collabCmd.UpdateTeamHandler
	DeleteTeam *collabCmd.DeleteTeamHandler

	// Invitation Commands
	SendInvitation   *collabCmd.SendInvitationHandler
	AcceptInvitation *collabCmd.AcceptInvitationHandler
	RejectInvitation *collabCmd.RejectInvitationHandler
	RevokeInvitation *collabCmd.RevokeInvitationHandler

	// Member Commands
	UpdateMemberRole *collabCmd.UpdateMemberRoleHandler
	RemoveMember     *collabCmd.RemoveMemberHandler
	LeaveTeam        *collabCmd.LeaveTeamHandler

	// Queries
	GetTeam           *collabQuery.GetTeamHandler
	ListMyTeams       *collabQuery.ListMyTeamsHandler
	ListMembers       *collabQuery.ListMembersHandler
	ListInvitations   *collabQuery.ListInvitationsHandler
	ListMyInvitations *collabQuery.ListMyInvitationsHandler
}

// InitializeBillingHandlers creates BillingHandlers with all dependencies.
func InitializeBillingHandlers(db *gorm.DB) (*BillingHandlers, error) {
	wire.Build(
		RepositorySet,
		BillingCommandSet,
		BillingQuerySet,
		wire.Struct(new(BillingHandlers), "*"),
	)
	return nil, nil
}

// InitializeOrderHandlers creates OrderHandlers with all dependencies.
func InitializeOrderHandlers(db *gorm.DB) (*OrderHandlers, error) {
	wire.Build(
		RepositorySet,
		OrderCommandSet,
		OrderQuerySet,
		wire.Struct(new(OrderHandlers), "*"),
	)
	return nil, nil
}

// InitializePaymentHandlers creates PaymentHandlers with all dependencies.
func InitializePaymentHandlers(db *gorm.DB) (*PaymentHandlers, error) {
	wire.Build(
		RepositorySet,
		PaymentCommandSet,
		PaymentQuerySet,
		wire.Struct(new(PaymentHandlers), "*"),
	)
	return nil, nil
}

// InitializeHSTHandlers creates all HTTP handlers with all dependencies.
func InitializeHSTHandlers(db *gorm.DB) (*HSTHandlers, error) {
	wire.Build(
		HSTSet,
		wire.Struct(new(HSTHandlers), "*"),
	)
	return nil, nil
}

// InitializeCollaborationHandlers creates CollaborationHandlers with all dependencies.
func InitializeCollaborationHandlers(db *gorm.DB) (*CollaborationHandlers, error) {
	wire.Build(
		RepositorySet,
		CollaborationCommandSet,
		CollaborationQuerySet,
		wire.Struct(new(CollaborationHandlers), "*"),
	)
	return nil, nil
}
