package outbound

import (
	"context"
	"io"
	"time"
)

// StoragePort defines object storage operations.
type StoragePort interface {
	// Put uploads an object to storage.
	Put(ctx context.Context, key string, reader io.Reader, size int64) error

	// Get retrieves an object from storage.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes an object from storage.
	Delete(ctx context.Context, key string) error

	// GetPresignedURL generates a presigned URL for temporary access.
	GetPresignedURL(ctx context.Context, key string, duration time.Duration) (string, error)
}
