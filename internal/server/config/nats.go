package config

import "time"

// Default configuration constants
const (
	// NATS connection defaults
	DefaultNATSHost = "localhost"
	DefaultNATSPort = "4222"

	// Timeout defaults
	DefaultRequestTimeout = 10 * time.Second
	DefaultDrainTimeout   = 30 * time.Second
	DefaultReconnectWait  = 2 * time.Second
	DefaultPingInterval   = 2 * time.Minute

	// Connection defaults
	DefaultMaxReconnects = -1 // Reconnect forever
	DefaultMaxPingsOut   = 2
)

// NATSConfig holds NATS-specific configuration
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

// Implement natz.Config interface methods
func (c *Config) Endpoint() string                 { return c.NATS.URL }
func (c *Config) NATSMaxReconnects() int           { return c.NATS.MaxReconnects }
func (c *Config) NATSReconnectWait() time.Duration { return c.NATS.ReconnectWait }
func (c *Config) NATSDrainTimeout() time.Duration  { return c.NATS.DrainTimeout }
func (c *Config) NATSPingInterval() time.Duration  { return c.NATS.PingInterval }
func (c *Config) NATSMaxPingsOut() int             { return c.NATS.MaxPingsOut }
func (c *Config) NATSClientName() string           { return c.NATS.ClientName }
