package collaboration

import "time"

// Config holds collaboration domain configuration.
type Config struct {
	// DefaultMemberLimit is the default member limit for new teams.
	DefaultMemberLimit int

	// InvitationExpiry is how long an invitation is valid.
	InvitationExpiry time.Duration

	// InvitationTokenLength is the length of invitation tokens.
	InvitationTokenLength int

	// BaseURL is the base URL for invitation links.
	BaseURL string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultMemberLimit:    5,
		InvitationExpiry:      7 * 24 * time.Hour, // 7 days
		InvitationTokenLength: 32,
		BaseURL:               "",
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.DefaultMemberLimit <= 0 {
		c.DefaultMemberLimit = 5
	}
	if c.InvitationExpiry <= 0 {
		c.InvitationExpiry = 7 * 24 * time.Hour
	}
	if c.InvitationTokenLength <= 0 {
		c.InvitationTokenLength = 32
	}
	return nil
}
