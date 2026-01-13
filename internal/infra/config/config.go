package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	HTTPClient HTTPClientConfig `mapstructure:"http_client"`
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
	AccessControl AccessControlConfig `mapstructure:"access_control"`
	AI         AIConfig         `mapstructure:"ai"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Media      MediaConfig      `mapstructure:"media"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Git        GitConfig        `mapstructure:"git"`
	Log        LogConfig        `mapstructure:"log"`
	Stripe     StripeConfig     `mapstructure:"stripe"`
	Alipay     AlipayConfig     `mapstructure:"alipay"`
	Wechat     WechatConfig     `mapstructure:"wechat"`
	Email      EmailConfig      `mapstructure:"email"`
	Features   FeaturesConfig   `mapstructure:"features"`
}

// FeaturesConfig holds feature flags for gradual rollout.
type FeaturesConfig struct {
	UseNewArchitecture bool `mapstructure:"use_new_architecture"` // Use hexagonal architecture (app_v2)
}

// AccessControlConfig holds privileged account configuration (admins/SRE).
type AccessControlConfig struct {
	AdminEmails  []string `mapstructure:"admin_emails"`
	SREEmails    []string `mapstructure:"sre_emails"`
	AdminUserIDs []string `mapstructure:"admin_user_ids"`
	SREUserIDs   []string `mapstructure:"sre_user_ids"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Address      string        `mapstructure:"address"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN returns the database connection string.
func (c *DatabaseConfig) DSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Database, c.SSLMode,
	)
	if c.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.Password)
	}
	return dsn
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// HTTPClientConfig holds HTTP client configuration for connection pooling.
type HTTPClientConfig struct {
	// Connection pool settings
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host"`
	MaxConnsPerHost     int           `mapstructure:"max_conns_per_host"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`

	// Timeout settings
	DialTimeout         time.Duration `mapstructure:"dial_timeout"`
	TLSHandshakeTimeout time.Duration `mapstructure:"tls_handshake_timeout"`
	ResponseTimeout     time.Duration `mapstructure:"response_timeout"`

	// Keep-alive settings
	KeepAlive time.Duration `mapstructure:"keep_alive"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	// Enabled enables/disables rate limiting.
	Enabled bool `mapstructure:"enabled"`
	// GlobalLimit is the global rate limit per IP per minute.
	GlobalLimit int `mapstructure:"global_limit"`
	// GlobalWindow is the global rate limit window.
	GlobalWindow time.Duration `mapstructure:"global_window"`
	// APILimit is the API rate limit per user per minute.
	APILimit int `mapstructure:"api_limit"`
	// APIWindow is the API rate limit window.
	APIWindow time.Duration `mapstructure:"api_window"`
	// IdempotencyTTL is the TTL for idempotency keys.
	IdempotencyTTL time.Duration `mapstructure:"idempotency_ttl"`
}

// AIConfig holds AI module configuration.
type AIConfig struct {
	HealthCheckInterval  time.Duration `mapstructure:"health_check_interval"`
	FailureThreshold     uint32        `mapstructure:"failure_threshold"`
	SuccessThreshold     uint32        `mapstructure:"success_threshold"`
	CircuitTimeout       time.Duration `mapstructure:"circuit_timeout"`
	TaskCleanupInterval  time.Duration `mapstructure:"task_cleanup_interval"`
	TaskRetentionPeriod  time.Duration `mapstructure:"task_retention_period"`
	MaxConcurrentTasks   int           `mapstructure:"max_concurrent_tasks"`
	EmbeddingCacheTTL    time.Duration `mapstructure:"embedding_cache_ttl"`

	// Account pool configuration
	AccountPoolScheduler     string        `mapstructure:"account_pool_scheduler"`      // round_robin, weighted, priority
	AccountPoolCacheTTL      time.Duration `mapstructure:"account_pool_cache_ttl"`
	AccountPoolEncryptionKey string        `mapstructure:"account_pool_encryption_key"` // Base64 encoded 32-byte key
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
	MasterKey          string        `mapstructure:"master_key"` // For API key encryption
	OAuth              OAuthConfig   `mapstructure:"oauth"`
}

// MediaConfig holds media pricing configuration.
type MediaConfig struct {
	// ImageUSDPerCredit is the pay-as-you-go price per image credit (1 credit = 1 generated image).
	ImageUSDPerCredit float64 `mapstructure:"image_usd_per_credit"`
	// VideoUSDPerMinute is the pay-as-you-go price per video minute.
	VideoUSDPerMinute float64 `mapstructure:"video_usd_per_minute"`
}

// OAuthConfig holds OAuth provider configurations.
type OAuthConfig struct {
	GitHub OAuthProviderConfig `mapstructure:"github"`
	Google OAuthProviderConfig `mapstructure:"google"`
}

// OAuthProviderConfig holds configuration for a single OAuth provider.
type OAuthProviderConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// StorageConfig holds object storage configuration.
type StorageConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
}

// GitConfig holds Git module configuration.
type GitConfig struct {
	RepoPrefix     string        `mapstructure:"repo_prefix"`       // R2 prefix for repos (default: "repos/")
	LFSPrefix      string        `mapstructure:"lfs_prefix"`        // R2 prefix for LFS objects (default: "lfs/")
	LFSURLExpiry   time.Duration `mapstructure:"lfs_url_expiry"`    // Presigned URL expiry (default: 1h)
	LFSMaxFileSize int64         `mapstructure:"lfs_max_file_size"` // Max LFS file size in bytes (default: 100GB)
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// StripeConfig holds Stripe payment configuration.
type StripeConfig struct {
	SecretKey      string `mapstructure:"secret_key"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	PublishableKey string `mapstructure:"publishable_key"`
}

// AlipayConfig holds Alipay payment configuration.
type AlipayConfig struct {
	AppID           string `mapstructure:"app_id"`
	PrivateKey      string `mapstructure:"private_key"`       // RSA2 private key (PEM format)
	AlipayPublicKey string `mapstructure:"alipay_public_key"` // Alipay public key (PEM format)
	IsProd          bool   `mapstructure:"is_prod"`
	NotifyURL       string `mapstructure:"notify_url"`
	ReturnURL       string `mapstructure:"return_url"`
}

// WechatConfig holds WeChat Pay configuration.
type WechatConfig struct {
	AppID                 string `mapstructure:"app_id"`
	MchID                 string `mapstructure:"mch_id"`                   // Merchant ID
	APIKeyV3              string `mapstructure:"api_key_v3"`               // APIv3 Key
	SerialNo              string `mapstructure:"serial_no"`                // Certificate serial number
	PrivateKey            string `mapstructure:"private_key"`              // Private key (PEM format)
	WechatPublicKeySerial string `mapstructure:"wechat_public_key_serial"` // Platform cert serial
	WechatPublicKey       string `mapstructure:"wechat_public_key"`        // Platform public key (PEM)
	IsProd                bool   `mapstructure:"is_prod"`
	NotifyURL             string `mapstructure:"notify_url"`
}

// EmailConfig holds email configuration.
type EmailConfig struct {
	Provider    string     `mapstructure:"provider"` // "smtp" or "noop"
	SMTP        SMTPConfig `mapstructure:"smtp"`
	FromAddress string     `mapstructure:"from_address"`
	FromName    string     `mapstructure:"from_name"`
	BaseURL     string     `mapstructure:"base_url"` // For verification links
}

// SMTPConfig holds SMTP configuration.
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// Load loads configuration from file and environment.
func Load() (*Config, error) {
	v := viper.New()

	// Set config file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/uniedit")

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// Config file not found, use defaults and env
	}

	// Read from environment variables
	v.SetEnvPrefix("UNIEDIT")
	v.AutomaticEnv()

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Override with environment variables for sensitive values
	if secret := os.Getenv("UNIEDIT_JWT_SECRET"); secret != "" {
		cfg.Auth.JWTSecret = secret
	}
	if masterKey := os.Getenv("UNIEDIT_MASTER_KEY"); masterKey != "" {
		cfg.Auth.MasterKey = masterKey
	}
	if password := os.Getenv("UNIEDIT_DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if password := os.Getenv("UNIEDIT_REDIS_PASSWORD"); password != "" {
		cfg.Redis.Password = password
	}
	if key := os.Getenv("UNIEDIT_STORAGE_SECRET_KEY"); key != "" {
		cfg.Storage.SecretAccessKey = key
	}
	// OAuth credentials from environment
	if clientID := os.Getenv("UNIEDIT_GITHUB_CLIENT_ID"); clientID != "" {
		cfg.Auth.OAuth.GitHub.ClientID = clientID
	}
	if clientSecret := os.Getenv("UNIEDIT_GITHUB_CLIENT_SECRET"); clientSecret != "" {
		cfg.Auth.OAuth.GitHub.ClientSecret = clientSecret
	}
	if clientID := os.Getenv("UNIEDIT_GOOGLE_CLIENT_ID"); clientID != "" {
		cfg.Auth.OAuth.Google.ClientID = clientID
	}
	if clientSecret := os.Getenv("UNIEDIT_GOOGLE_CLIENT_SECRET"); clientSecret != "" {
		cfg.Auth.OAuth.Google.ClientSecret = clientSecret
	}
	// Stripe credentials from environment
	if secretKey := os.Getenv("UNIEDIT_STRIPE_SECRET_KEY"); secretKey != "" {
		cfg.Stripe.SecretKey = secretKey
	}
	if webhookSecret := os.Getenv("UNIEDIT_STRIPE_WEBHOOK_SECRET"); webhookSecret != "" {
		cfg.Stripe.WebhookSecret = webhookSecret
	}
	// SMTP credentials from environment
	if password := os.Getenv("UNIEDIT_SMTP_PASSWORD"); password != "" {
		cfg.Email.SMTP.Password = password
	}
	// Alipay credentials from environment
	if appID := os.Getenv("UNIEDIT_ALIPAY_APP_ID"); appID != "" {
		cfg.Alipay.AppID = appID
	}
	if privateKey := os.Getenv("UNIEDIT_ALIPAY_PRIVATE_KEY"); privateKey != "" {
		cfg.Alipay.PrivateKey = privateKey
	}
	if publicKey := os.Getenv("UNIEDIT_ALIPAY_PUBLIC_KEY"); publicKey != "" {
		cfg.Alipay.AlipayPublicKey = publicKey
	}
	// WeChat credentials from environment
	if appID := os.Getenv("UNIEDIT_WECHAT_APP_ID"); appID != "" {
		cfg.Wechat.AppID = appID
	}
	if mchID := os.Getenv("UNIEDIT_WECHAT_MCH_ID"); mchID != "" {
		cfg.Wechat.MchID = mchID
	}
	if apiKey := os.Getenv("UNIEDIT_WECHAT_API_KEY_V3"); apiKey != "" {
		cfg.Wechat.APIKeyV3 = apiKey
	}
	if privateKey := os.Getenv("UNIEDIT_WECHAT_PRIVATE_KEY"); privateKey != "" {
		cfg.Wechat.PrivateKey = privateKey
	}

	// Access control from environment (comma-separated lists).
	if s := os.Getenv("UNIEDIT_ADMIN_EMAILS"); s != "" {
		cfg.AccessControl.AdminEmails = parseCommaSeparatedList(s)
	}
	if s := os.Getenv("UNIEDIT_SRE_EMAILS"); s != "" {
		cfg.AccessControl.SREEmails = parseCommaSeparatedList(s)
	}
	if s := os.Getenv("UNIEDIT_ADMIN_USER_IDS"); s != "" {
		cfg.AccessControl.AdminUserIDs = parseCommaSeparatedList(s)
	}
	if s := os.Getenv("UNIEDIT_SRE_USER_IDS"); s != "" {
		cfg.AccessControl.SREUserIDs = parseCommaSeparatedList(s)
	}

	return &cfg, nil
}

func parseCommaSeparatedList(s string) []string {
	if s == "" {
		return nil
	}

	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			start = i + 1
			part = trimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func trimSpace(s string) string {
	// strings.TrimSpace without importing strings in config hot path.
	start := 0
	for start < len(s) {
		switch s[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			goto leftDone
		}
	}
leftDone:
	end := len(s)
	for end > start {
		switch s[end-1] {
		case ' ', '\t', '\n', '\r':
			end--
		default:
			goto rightDone
		}
	}
rightDone:
	return s[start:end]
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 120*time.Second)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.database", "uniedit")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", time.Hour)
	v.SetDefault("database.conn_max_idle_time", 30*time.Minute)

	// Redis defaults
	v.SetDefault("redis.address", "localhost:6379")
	v.SetDefault("redis.db", 0)

	// HTTP client defaults
	v.SetDefault("http_client.max_idle_conns", 100)
	v.SetDefault("http_client.max_idle_conns_per_host", 20)
	v.SetDefault("http_client.max_conns_per_host", 50)
	v.SetDefault("http_client.idle_conn_timeout", 90*time.Second)
	v.SetDefault("http_client.dial_timeout", 30*time.Second)
	v.SetDefault("http_client.tls_handshake_timeout", 10*time.Second)
	v.SetDefault("http_client.response_timeout", 120*time.Second)
	v.SetDefault("http_client.keep_alive", 30*time.Second)

	// Rate limit defaults
	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.global_limit", 100)
	v.SetDefault("rate_limit.global_window", time.Minute)
	v.SetDefault("rate_limit.api_limit", 60)
	v.SetDefault("rate_limit.api_window", time.Minute)
	v.SetDefault("rate_limit.idempotency_ttl", 24*time.Hour)

	// Access control defaults
	v.SetDefault("access_control.admin_emails", []string{})
	v.SetDefault("access_control.sre_emails", []string{})
	v.SetDefault("access_control.admin_user_ids", []string{})
	v.SetDefault("access_control.sre_user_ids", []string{})

	// AI defaults
	v.SetDefault("ai.health_check_interval", 30*time.Second)
	v.SetDefault("ai.failure_threshold", 5)
	v.SetDefault("ai.success_threshold", 2)
	v.SetDefault("ai.circuit_timeout", 60*time.Second)
	v.SetDefault("ai.task_cleanup_interval", 5*time.Minute)
	v.SetDefault("ai.task_retention_period", 24*time.Hour)
	v.SetDefault("ai.max_concurrent_tasks", 100)
	v.SetDefault("ai.embedding_cache_ttl", 24*time.Hour)
	v.SetDefault("ai.account_pool_scheduler", "round_robin")
	v.SetDefault("ai.account_pool_cache_ttl", 5*time.Minute)

	// Auth defaults
	v.SetDefault("auth.access_token_expiry", 15*time.Minute)
	v.SetDefault("auth.refresh_token_expiry", 7*24*time.Hour)

	// Media defaults (pricing)
	v.SetDefault("media.image_usd_per_credit", 0.04)
	v.SetDefault("media.video_usd_per_minute", 0.20)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")

	// Email defaults
	v.SetDefault("email.provider", "noop")
	v.SetDefault("email.smtp.port", 587)
	v.SetDefault("email.from_name", "UniEdit")

	// Git defaults
	v.SetDefault("git.repo_prefix", "repos/")
	v.SetDefault("git.lfs_prefix", "lfs/")
	v.SetDefault("git.lfs_url_expiry", time.Hour)
	v.SetDefault("git.lfs_max_file_size", 100*1024*1024*1024) // 100GB

	// Feature flags defaults
	v.SetDefault("features.use_new_architecture", true)
}
