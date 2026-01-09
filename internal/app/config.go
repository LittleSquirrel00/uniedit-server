package app

import (
	"github.com/uniedit/server/internal/infra/config"
)

// LoadConfig loads application configuration.
func LoadConfig() (*config.Config, error) {
	return config.Load()
}
