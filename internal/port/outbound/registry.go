package outbound

// OutboundPorts holds all outbound port implementations.
// Outbound ports define how the application interacts with external systems.
type OutboundPorts struct {
	// Generic infrastructure ports
	Database DatabasePort
	Cache    CachePort
	Storage  StoragePort
	Message  MessagePort

	// User ports (Phase 1)
	// UserDB        UserDatabasePort
	// ProfileDB     ProfileDatabasePort
	// PrefsDB       PreferencesDatabasePort
	// AvatarStorage AvatarStoragePort

	// Auth ports (Phase 2)
	// AuthDB     AuthDatabasePort
	// SessionDB  SessionDatabasePort
	// OAuthPort  OAuthPort

	// Billing ports (Phase 3)
	// BillingDB BillingDatabasePort

	// Payment ports (Phase 5)
	// PaymentGateway PaymentGatewayPort
}

// NewOutboundPorts creates outbound ports with infrastructure dependencies.
// func NewOutboundPorts(db *gorm.DB, redis *redis.Client, s3 *s3.Client) *OutboundPorts {
// 	return &OutboundPorts{
// 		// Initialize ports as they are migrated
// 	}
// }
