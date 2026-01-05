package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	AI       AIConfig       `mapstructure:"ai"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Log      LogConfig      `mapstructure:"log"`
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
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AIConfig holds AI module configuration.
type AIConfig struct {
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	FailureThreshold    uint32        `mapstructure:"failure_threshold"`
	SuccessThreshold    uint32        `mapstructure:"success_threshold"`
	CircuitTimeout      time.Duration `mapstructure:"circuit_timeout"`
	TaskCleanupInterval time.Duration `mapstructure:"task_cleanup_interval"`
	TaskRetentionPeriod time.Duration `mapstructure:"task_retention_period"`
	MaxConcurrentTasks  int           `mapstructure:"max_concurrent_tasks"`
	EmbeddingCacheTTL   time.Duration `mapstructure:"embedding_cache_ttl"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret          string        `mapstructure:"jwt_secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
}

// StorageConfig holds object storage configuration.
type StorageConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
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
	if password := os.Getenv("UNIEDIT_DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if password := os.Getenv("UNIEDIT_REDIS_PASSWORD"); password != "" {
		cfg.Redis.Password = password
	}
	if key := os.Getenv("UNIEDIT_STORAGE_SECRET_KEY"); key != "" {
		cfg.Storage.SecretAccessKey = key
	}

	return &cfg, nil
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

	// AI defaults
	v.SetDefault("ai.health_check_interval", 30*time.Second)
	v.SetDefault("ai.failure_threshold", 5)
	v.SetDefault("ai.success_threshold", 2)
	v.SetDefault("ai.circuit_timeout", 60*time.Second)
	v.SetDefault("ai.task_cleanup_interval", 5*time.Minute)
	v.SetDefault("ai.task_retention_period", 24*time.Hour)
	v.SetDefault("ai.max_concurrent_tasks", 100)
	v.SetDefault("ai.embedding_cache_ttl", 24*time.Hour)

	// Auth defaults
	v.SetDefault("auth.access_token_expiry", 15*time.Minute)
	v.SetDefault("auth.refresh_token_expiry", 7*24*time.Hour)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}
