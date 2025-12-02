package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Default configuration constants tuned for SDK clients.
const (
	DefaultNATSHost = "localhost"
	DefaultNATSPort = "4222"

	DefaultRequestTimeout = 10 * time.Second
	DefaultDrainTimeout   = 30 * time.Second
	DefaultReconnectWait  = 2 * time.Second
	DefaultPingInterval   = 2 * time.Minute

	DefaultMaxReconnects = -1 // reconnect forever
	DefaultMaxPingsOut   = 2
)

// NATSConfig holds NATS-specific configuration knobs for the SDK.
type NATSConfig struct {
	URL           string        `json:"url"             env:"URL"`
	Host          string        `json:"host"            env:"HOST"`
	Port          string        `json:"port"            env:"PORT"`
	MaxReconnects int           `json:"max_reconnects"  env:"MAX_RECONNECTS"`
	ReconnectWait time.Duration `json:"reconnect_wait"  env:"RECONNECT_WAIT"`
	DrainTimeout  time.Duration `json:"drain_timeout"   env:"DRAIN_TIMEOUT"`
	PingInterval  time.Duration `json:"ping_interval"   env:"PING_INTERVAL"`
	MaxPingsOut   int           `json:"max_pings_out"   env:"MAX_PINGS_OUT"`
	ClientName    string        `json:"client_name"     env:"CLIENT_NAME"`
}

// TimeoutConfig encapsulates SDK timeout values.
type TimeoutConfig struct {
	RequestTimeout time.Duration `json:"request_timeout" env:"REQUEST_TIMEOUT"`
}

// Config is the public SDK configuration users can construct or load from env.
type Config struct {
	NATS     NATSConfig    `json:"nats" envPrefix:"NATS_"`
	Timeouts TimeoutConfig `json:"timeouts" envPrefix:"TIMEOUTS_"`
}

// Load loads configuration from environment variables applying defaults.
func Load() (*Config, error) {
	cfg := Config{
		NATS: NATSConfig{
			Host:          DefaultNATSHost,
			Port:          DefaultNATSPort,
			MaxReconnects: DefaultMaxReconnects,
			ReconnectWait: DefaultReconnectWait,
			DrainTimeout:  DefaultDrainTimeout,
			PingInterval:  DefaultPingInterval,
			MaxPingsOut:   DefaultMaxPingsOut,
			ClientName:    "durablefuture-sdk",
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: DefaultRequestTimeout,
		},
	}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	if cfg.NATS.URL == "" {
		cfg.NATS.URL = fmt.Sprintf("nats://%s:%s", cfg.NATS.Host, cfg.NATS.Port)
	}
	return &cfg, nil
}

// Interface implementation for internal JetStream connection.
func (c *Config) Endpoint() string                 { return c.NATS.URL }
func (c *Config) NATSMaxReconnects() int           { return c.NATS.MaxReconnects }
func (c *Config) NATSReconnectWait() time.Duration { return c.NATS.ReconnectWait }
func (c *Config) NATSDrainTimeout() time.Duration  { return c.NATS.DrainTimeout }
func (c *Config) NATSPingInterval() time.Duration  { return c.NATS.PingInterval }
func (c *Config) NATSMaxPingsOut() int             { return c.NATS.MaxPingsOut }
func (c *Config) NATSClientName() string           { return c.NATS.ClientName }
