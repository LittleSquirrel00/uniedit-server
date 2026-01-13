package media

import "time"

// Config holds media domain configuration.
type Config struct {
	// VideoPollInterval is the interval for polling video status.
	VideoPollInterval time.Duration

	// MaxConcurrentTasks is the maximum number of concurrent tasks.
	MaxConcurrentTasks int

	// TaskTimeout is the timeout for a single task.
	TaskTimeout time.Duration

	// ImageUSDPerCredit is the pay-as-you-go price per image credit (1 credit = 1 generated image).
	ImageUSDPerCredit float64

	// VideoUSDPerMinute is the pay-as-you-go price per video minute (ceil).
	VideoUSDPerMinute float64
}

// DefaultConfig returns default media configuration.
func DefaultConfig() *Config {
	return &Config{
		VideoPollInterval:  5 * time.Second,
		MaxConcurrentTasks: 10,
		TaskTimeout:        30 * time.Minute,
		ImageUSDPerCredit:  0,
		VideoUSDPerMinute:  0,
	}
}
