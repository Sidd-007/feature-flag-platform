package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Database configurations
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`

	// Message broker configuration
	NATS NATSConfig `mapstructure:"nats"`

	// Observability
	Observability ObservabilityConfig `mapstructure:"observability"`

	// Authentication
	Auth AuthConfig `mapstructure:"auth"`

	// Feature Flag specific
	FeatureFlags FeatureFlagConfig `mapstructure:"feature_flags"`

	// Service-specific configurations
	ControlPlane    ControlPlaneConfig    `mapstructure:"control_plane"`
	EdgeEvaluator   EdgeEvaluatorConfig   `mapstructure:"edge_evaluator"`
	EventIngestor   EventIngestorConfig   `mapstructure:"event_ingestor"`
	AnalyticsEngine AnalyticsEngineConfig `mapstructure:"analytics_engine"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	Environment     string        `mapstructure:"environment"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Database     string        `mapstructure:"database"`
	Username     string        `mapstructure:"username"`
	Password     string        `mapstructure:"password"`
	SSLMode      string        `mapstructure:"ssl_mode"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	Database     int           `mapstructure:"database"`
	PoolSize     int           `mapstructure:"pool_size"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// NATSConfig holds NATS connection configuration
type NATSConfig struct {
	URL             string        `mapstructure:"url"`
	MaxReconnect    int           `mapstructure:"max_reconnect"`
	ReconnectWait   time.Duration `mapstructure:"reconnect_wait"`
	Timeout         time.Duration `mapstructure:"timeout"`
	JetStreamDomain string        `mapstructure:"jetstream_domain"`
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Structured bool   `mapstructure:"structured"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret        string        `mapstructure:"jwt_secret"`
	JWTExpiry        time.Duration `mapstructure:"jwt_expiry"`
	OIDCIssuer       string        `mapstructure:"oidc_issuer"`
	OIDCClientID     string        `mapstructure:"oidc_client_id"`
	OIDCClientSecret string        `mapstructure:"oidc_client_secret"`
	BCryptCost       int           `mapstructure:"bcrypt_cost"`
}

// FeatureFlagConfig holds feature flag specific configuration
type FeatureFlagConfig struct {
	ConfigCacheTTL        time.Duration `mapstructure:"config_cache_ttl"`
	ConfigRefreshInterval time.Duration `mapstructure:"config_refresh_interval"`
	EvaluationTimeout     time.Duration `mapstructure:"evaluation_timeout"`
	MaxRulesPerFlag       int           `mapstructure:"max_rules_per_flag"`
	MaxSegmentsPerEnv     int           `mapstructure:"max_segments_per_env"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set up environment variable handling
	v.SetEnvPrefix("FF")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/feature-flags")

	// Read config file if it exists (not required)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Workaround: manually set config values if Viper found them but unmarshaling didn't work
	if config.Auth.JWTSecret == "" && v.GetString("auth.jwt_secret") != "" {
		config.Auth.JWTSecret = v.GetString("auth.jwt_secret")
	}

	// Fix for Control Plane URL
	if config.ControlPlane.URL == "" && v.GetString("control_plane.url") != "" {
		config.ControlPlane.URL = v.GetString("control_plane.url")
	}

	// Fix for Edge Evaluator API Key
	if config.EdgeEvaluator.APIKey == "" && v.GetString("edge_evaluator.api_key") != "" {
		config.EdgeEvaluator.APIKey = v.GetString("edge_evaluator.api_key")
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "30s")
	v.SetDefault("server.environment", "development")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.database", "feature_flags")
	v.SetDefault("database.username", "postgres")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.max_lifetime", "5m")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.database", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.max_reconnect", 10)
	v.SetDefault("nats.reconnect_wait", "2s")
	v.SetDefault("nats.timeout", "5s")

	// Observability defaults
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.path", "/metrics")
	v.SetDefault("observability.metrics.port", 9090)
	v.SetDefault("observability.tracing.enabled", false)
	v.SetDefault("observability.tracing.sample_rate", 0.1)
	v.SetDefault("observability.logging.level", "info")
	v.SetDefault("observability.logging.format", "json")
	v.SetDefault("observability.logging.output", "stdout")
	v.SetDefault("observability.logging.structured", true)

	// Auth defaults
	v.SetDefault("auth.jwt_expiry", "24h")
	v.SetDefault("auth.bcrypt_cost", 12)

	// Feature flag defaults
	v.SetDefault("feature_flags.config_cache_ttl", "5m")
	v.SetDefault("feature_flags.config_refresh_interval", "30s")
	v.SetDefault("feature_flags.evaluation_timeout", "100ms")
	v.SetDefault("feature_flags.max_rules_per_flag", 50)
	v.SetDefault("feature_flags.max_segments_per_env", 100)

	// Service-specific defaults
	v.SetDefault("control_plane.url", "http://localhost:8080")
	v.SetDefault("edge_evaluator.api_key", "")
	v.SetDefault("edge_evaluator.poll_interval", "30s")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	if c.NATS.URL == "" {
		return fmt.Errorf("NATS URL is required")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// ControlPlaneConfig holds Control Plane specific configuration
type ControlPlaneConfig struct {
	URL string `mapstructure:"url"`
}

// EdgeEvaluatorConfig holds Edge Evaluator specific configuration
type EdgeEvaluatorConfig struct {
	APIKey       string        `mapstructure:"api_key"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
}

// EventIngestorConfig holds Event Ingestor specific configuration
type EventIngestorConfig struct {
	URL          string        `mapstructure:"url"`
	APIKey       string        `mapstructure:"api_key"`
	BatchSize    int           `mapstructure:"batch_size"`
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`
}

// AnalyticsEngineConfig holds Analytics Engine specific configuration
type AnalyticsEngineConfig struct {
	QueryTimeout time.Duration `mapstructure:"query_timeout"`
}
