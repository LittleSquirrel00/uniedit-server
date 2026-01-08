package query

import "context"

// Handler is the interface for query handlers.
// Queries are read operations that don't modify state.
type Handler[Q any, R any] interface {
	Handle(ctx context.Context, query Q) (R, error)
}

// HandlerFunc is a function adapter for Handler interface.
type HandlerFunc[Q any, R any] func(ctx context.Context, query Q) (R, error)

// Handle implements Handler interface.
func (f HandlerFunc[Q, R]) Handle(ctx context.Context, query Q) (R, error) {
	return f(ctx, query)
}

// Bus dispatches queries to their handlers.
type Bus interface {
	Dispatch(ctx context.Context, query any) (any, error)
}
