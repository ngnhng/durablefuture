package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	env "github.com/caarlos0/env/v11"
	"github.com/ngnhng/durablefuture/internal/server/types"
)

// Config holds the complete application configuration
type Config struct {
	Service  string        `json:"service_name" env:"APP_NAME"    envDefault:"df"`
	Version  string        `json:"version"      env:"VERSION"     envDefault:"v0.1.0-alpha1"`
	Mode     types.Mode    `json:"mode"         env:"MODE"        envDefault:"debug"`
	NATS     NATSConfig    `json:"nats"         envPrefix:"NATS_"`
	Server   ServerConfig  `json:"server"       envPrefix:"SERVER_"`
	Timeouts TimeoutConfig `json:"timeouts"     envPrefix:"TIMEOUTS_"`
	Logger   LoggerConfig  `json:"logger"       envPrefix:"LOG_"`
}

type ServerConfig struct {
	Host string `json:"host" env:"HOST" envDefault:"localhost"`
	Port string `json:"port" env:"PORT" envDefault:"8080"`
}

// TimeoutConfig holds timeout-related configuration
type TimeoutConfig struct {
	RequestTimeout time.Duration `json:"request_timeout" env:"REQUEST_TIMEOUT"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Service == "" {
		return errors.New("service name is required")
	}

	if c.Version == "" {
		return errors.New("version is required")
	}

	// Validate NATS config
	if c.NATS.Host == "" {
		return errors.New("NATS host is required")
	}

	if c.NATS.Port == "" {
		return errors.New("NATS port is required")
	}

	if _, err := strconv.Atoi(c.NATS.Port); err != nil {
		return fmt.Errorf("invalid NATS port '%s': must be a number", c.NATS.Port)
	}

	if c.NATS.URL == "" {
		return errors.New("NATS URL is required")
	}

	if c.NATS.MaxReconnects < -1 {
		return errors.New("NATS max reconnects must be >= -1")
	}

	if c.NATS.ReconnectWait <= 0 {
		return errors.New("NATS reconnect wait must be positive")
	}

	if c.NATS.DrainTimeout <= 0 {
		return errors.New("NATS drain timeout must be positive")
	}

	// Validate Server config
	if c.Server.Host == "" {
		return errors.New("server host is required")
	}

	if c.Server.Port == "" {
		return errors.New("server port is required")
	}

	if _, err := strconv.Atoi(c.Server.Port); err != nil {
		return fmt.Errorf("invalid server port '%s': must be a number", c.Server.Port)
	}

	return nil
}

func LoadConfig() (*Config, error) {
	cfg := Config{
		NATS: NATSConfig{
			Host:          DefaultNATSHost,
			Port:          DefaultNATSPort,
			MaxReconnects: DefaultMaxReconnects,
			ReconnectWait: DefaultReconnectWait,
			DrainTimeout:  DefaultDrainTimeout,
			PingInterval:  DefaultPingInterval,
			MaxPingsOut:   DefaultMaxPingsOut,
			ClientName:    "durablefuture",
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: DefaultRequestTimeout,
		},
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.NATS.URL == "" {
		cfg.NATS.URL = fmt.Sprintf("nats://%s:%s", cfg.NATS.Host, cfg.NATS.Port)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func (c *Config) ServiceName() string {
	return c.Service
}

func (c *Config) GetVersion() string {
	return c.Version
}
