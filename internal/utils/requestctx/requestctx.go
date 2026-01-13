package requestctx

import "context"

type ctxKey int

const (
	requestIDKey ctxKey = iota
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		return context.WithValue(context.Background(), requestIDKey, requestID)
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
