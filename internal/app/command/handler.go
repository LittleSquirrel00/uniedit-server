package command

import "context"

// Handler is the interface for command handlers.
// Commands are write operations that modify state.
type Handler[C any, R any] interface {
	Handle(ctx context.Context, cmd C) (R, error)
}

// HandlerFunc is a function adapter for Handler interface.
type HandlerFunc[C any, R any] func(ctx context.Context, cmd C) (R, error)

// Handle implements Handler interface.
func (f HandlerFunc[C, R]) Handle(ctx context.Context, cmd C) (R, error) {
	return f(ctx, cmd)
}

// NoResult is used for commands that don't return a result.
type NoResult struct{}

// Bus dispatches commands to their handlers.
type Bus interface {
	Dispatch(ctx context.Context, cmd any) (any, error)
}
