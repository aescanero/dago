package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the DA Orchestrator
type Config struct {
	// Server configuration
	HTTPPort int    `env:"DAGO_HTTP_PORT" envDefault:"8080"`
	GRPCPort int    `env:"DAGO_GRPC_PORT" envDefault:"9090"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	// Redis configuration
	Redis RedisConfig

	// Timeouts
	Timeouts TimeoutConfig
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASS"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`

	// Connection pool settings
	PoolSize     int           `env:"REDIS_POOL_SIZE" envDefault:"10"`
	MinIdleConns int           `env:"REDIS_MIN_IDLE_CONNS" envDefault:"2"`
	MaxRetries   int           `env:"REDIS_MAX_RETRIES" envDefault:"3"`
	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" envDefault:"5s"`
	ReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT" envDefault:"3s"`
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" envDefault:"3s"`
}

// TimeoutConfig holds various timeout configurations
type TimeoutConfig struct {
	GraphExecutionTimeout time.Duration `env:"TIMEOUT_GRAPH_EXECUTION" envDefault:"3600s"` // 1 hour
	NodeExecutionTimeout  time.Duration `env:"TIMEOUT_NODE_EXECUTION" envDefault:"300s"`   // 5 minutes
	ShutdownTimeout       time.Duration `env:"TIMEOUT_SHUTDOWN" envDefault:"30s"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server ports
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.HTTPPort)
	}
	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPCPort)
	}

	// Validate Redis config
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis address is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

// GetHTTPAddr returns the HTTP server address
func (c *Config) GetHTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

// GetGRPCAddr returns the gRPC server address
func (c *Config) GetGRPCAddr() string {
	return fmt.Sprintf(":%d", c.GRPCPort)
}
