// Copyright 2025 Nguyen-Nhat Nguyen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

// Default configuration constants
const (
	// NATS connection defaults
	DefaultNATSHost = "localhost"
	DefaultNATSPort = "4222"
	
	// Timeout defaults
	DefaultRequestTimeout    = 10 * time.Second
	DefaultDrainTimeout      = 30 * time.Second
	DefaultReconnectWait     = 2 * time.Second
	DefaultPingInterval      = 2 * time.Minute
	
	// Connection defaults
	DefaultMaxReconnects = -1 // Reconnect forever
	DefaultMaxPingsOut   = 2
	
	// Activity simulation defaults
	DefaultChargeProcessingTime = 10 * time.Second
	DefaultShipProcessingTime   = 3 * time.Second
	DefaultDelayedActivityTime  = 1 * time.Second
	DefaultShippingDays         = 3
)

// Config holds the complete application configuration
type Config struct {
	NATS     NATSConfig     `json:"nats"`
	Server   ServerConfig   `json:"server"`
	Timeouts TimeoutConfig  `json:"timeouts"`
	Activity ActivityConfig `json:"activity"`
}

// NATSConfig holds NATS-specific configuration
type NATSConfig struct {
	URL           string        `json:"url"`
	Host          string        `json:"host"`
	Port          string        `json:"port"`
	MaxReconnects int           `json:"max_reconnects"`
	ReconnectWait time.Duration `json:"reconnect_wait"`
	DrainTimeout  time.Duration `json:"drain_timeout"`
	PingInterval  time.Duration `json:"ping_interval"`
	MaxPingsOut   int           `json:"max_pings_out"`
	Name          string        `json:"name"`
}

// ServerConfig holds server-specific configuration  
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// TimeoutConfig holds timeout-related configuration
type TimeoutConfig struct {
	RequestTimeout time.Duration `json:"request_timeout"`
}

// ActivityConfig holds activity simulation configuration
type ActivityConfig struct {
	ChargeProcessingTime time.Duration `json:"charge_processing_time"`
	ShipProcessingTime   time.Duration `json:"ship_processing_time"`
	DelayedActivityTime  time.Duration `json:"delayed_activity_time"`
	ShippingDays         int           `json:"shipping_days"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	cfg := &Config{
		NATS: NATSConfig{
			URL:           getEnvOrDefault("NATS_URL", buildDefaultNATSURL()),
			Host:          getEnvOrDefault("NATS_HOST", DefaultNATSHost),
			Port:          getEnvOrDefault("NATS_PORT", DefaultNATSPort),
			MaxReconnects: getEnvAsIntOrDefault("NATS_MAX_RECONNECTS", DefaultMaxReconnects),
			ReconnectWait: getEnvAsDurationOrDefault("NATS_RECONNECT_WAIT", DefaultReconnectWait),
			DrainTimeout:  getEnvAsDurationOrDefault("NATS_DRAIN_TIMEOUT", DefaultDrainTimeout),
			PingInterval:  getEnvAsDurationOrDefault("NATS_PING_INTERVAL", DefaultPingInterval),
			MaxPingsOut:   getEnvAsIntOrDefault("NATS_MAX_PINGS_OUT", DefaultMaxPingsOut),
			Name:          getEnvOrDefault("NATS_CLIENT_NAME", "durablefuture-client"),
		},
		Server: ServerConfig{
			Host: getEnvOrDefault("SERVER_HOST", DefaultNATSHost),
			Port: getEnvOrDefault("SERVER_PORT", DefaultNATSPort),
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: getEnvAsDurationOrDefault("REQUEST_TIMEOUT", DefaultRequestTimeout),
		},
		Activity: ActivityConfig{
			ChargeProcessingTime: getEnvAsDurationOrDefault("CHARGE_PROCESSING_TIME", DefaultChargeProcessingTime),
			ShipProcessingTime:   getEnvAsDurationOrDefault("SHIP_PROCESSING_TIME", DefaultShipProcessingTime),
			DelayedActivityTime:  getEnvAsDurationOrDefault("DELAYED_ACTIVITY_TIME", DefaultDelayedActivityTime),
			ShippingDays:         getEnvAsIntOrDefault("SHIPPING_DAYS", DefaultShippingDays),
		},
	}

	// If NATS_URL is not set, construct it from host and port
	if cfg.NATS.URL == buildDefaultNATSURL() {
		cfg.NATS.URL = fmt.Sprintf("nats://%s:%s", cfg.NATS.Host, cfg.NATS.Port)
	}

	return cfg
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.NATS.URL == "" {
		return fmt.Errorf("NATS URL cannot be empty")
	}
	if c.NATS.Host == "" {
		return fmt.Errorf("NATS host cannot be empty")
	}
	if c.NATS.Port == "" {
		return fmt.Errorf("NATS port cannot be empty")
	}
	if c.Timeouts.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}
	return nil
}

// GetNATSURL returns the NATS connection URL
func (c *Config) GetNATSURL() string {
	return c.NATS.URL
}

// buildDefaultNATSURL builds the default NATS URL
func buildDefaultNATSURL() string {
	if url := os.Getenv("NATS_URL"); url != "" {
		return url
	}
	return nats.DefaultURL
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}