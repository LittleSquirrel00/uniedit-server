package outbound

import (
	"context"

	"gorm.io/gorm"
)

// DatabasePort defines generic database operations.
type DatabasePort interface {
	// DB returns the underlying database connection.
	DB() *gorm.DB

	// Transaction executes a function within a database transaction.
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}
