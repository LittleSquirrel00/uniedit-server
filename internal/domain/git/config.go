package git

import "time"

// Config holds Git domain configuration.
type Config struct {
	// RepoPrefix is the prefix for repository storage paths.
	RepoPrefix string

	// LFSPrefix is the prefix for LFS object storage paths.
	LFSPrefix string

	// MaxLFSFileSize is the maximum allowed LFS file size in bytes.
	MaxLFSFileSize int64

	// PresignedURLExpiry is the default expiry for presigned URLs.
	PresignedURLExpiry time.Duration

	// DefaultBranch is the default branch name for new repositories.
	DefaultBranch string

	// BaseURL is the base URL for generating clone URLs.
	BaseURL string
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		RepoPrefix:         "repos/",
		LFSPrefix:          "lfs/",
		MaxLFSFileSize:     2 * 1024 * 1024 * 1024, // 2GB
		PresignedURLExpiry: 1 * time.Hour,
		DefaultBranch:      "main",
		BaseURL:            "",
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.RepoPrefix == "" {
		c.RepoPrefix = "repos/"
	}
	if c.LFSPrefix == "" {
		c.LFSPrefix = "lfs/"
	}
	if c.MaxLFSFileSize <= 0 {
		c.MaxLFSFileSize = 2 * 1024 * 1024 * 1024
	}
	if c.PresignedURLExpiry <= 0 {
		c.PresignedURLExpiry = 1 * time.Hour
	}
	if c.DefaultBranch == "" {
		c.DefaultBranch = "main"
	}
	return nil
}
