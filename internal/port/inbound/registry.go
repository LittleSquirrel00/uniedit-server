package inbound

// InboundPorts holds all inbound port implementations.
// Inbound ports define how external actors interact with the application.
type InboundPorts struct {
	// HTTP handlers
	// UserHTTP    UserHttpPort    // Phase 1
	// AuthHTTP    AuthHttpPort    // Phase 2
	// BillingHTTP BillingHttpPort // Phase 3

	// Admin handlers
	// UserAdmin UserAdminPort // Phase 1

	// gRPC handlers (future)
	// ...

	// CLI commands (future)
	// ...
}

// NewInboundPorts creates inbound ports with domain services.
// func NewInboundPorts(domain *domain.Domain) *InboundPorts {
// 	return &InboundPorts{
// 		// Initialize ports as they are migrated
// 	}
// }
