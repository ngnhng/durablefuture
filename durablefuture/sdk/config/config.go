// Copyright 2025 Nguyen Nhat Nguyen
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
	"time"

	env "github.com/caarlos0/env/v11"
)

// Config is the public SDK configuration users can construct or load from env.
type Config struct {
	NATS     NATSConfig    `json:"nats" envPrefix:"NATS_"`
	Timeouts TimeoutConfig `json:"timeouts" envPrefix:"TIMEOUTS_"`
}

type NATSConfig struct {
	URL           string        `json:"url"            env:"URL"`
	Host          string        `json:"host"           env:"HOST"           envDefault:"localhost"`
	Port          string        `json:"port"           env:"PORT"           envDefault:"4222"`
	MaxReconnects int           `json:"max_reconnects" env:"MAX_RECONNECTS" envDefault:"-1"`
	ReconnectWait time.Duration `json:"reconnect_wait" env:"RECONNECT_WAIT" envDefault:"2s"`
	DrainTimeout  time.Duration `json:"drain_timeout"  env:"DRAIN_TIMEOUT"  envDefault:"30s"`
	PingInterval  time.Duration `json:"ping_interval"  env:"PING_INTERVAL"  envDefault:"2m"`
	MaxPingsOut   int           `json:"max_pings_out"  env:"MAX_PINGS_OUT"  envDefault:"2"`
	ClientName    string        `json:"client_name"    env:"CLIENT_NAME"    envDefault:"durablefuture"`
}

type TimeoutConfig struct {
	// RequestTimeout is used by the SDK client when issuing RPC-like command requests
	// (e.g., starting a workflow) to wait for the manager's response.
	// If unset (zero) the client will defensively fall back to 10s.
	RequestTimeout time.Duration `json:"request_timeout" env:"REQUEST_TIMEOUT" envDefault:"10s"`
}

// Load loads configuration from environment variables applying defaults.
func Load() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	// Compose URL if not provided explicitly.
	if cfg.NATS.URL == "" {
		cfg.NATS.URL = fmt.Sprintf("nats://%s:%s", cfg.NATS.Host, cfg.NATS.Port)
	}
	return &cfg, nil
}

// Interface implementation for internal/natz.Config
func (c *Config) NATSURL() string                  { return c.NATS.URL }
func (c *Config) NATSMaxReconnects() int           { return c.NATS.MaxReconnects }
func (c *Config) NATSReconnectWait() time.Duration { return c.NATS.ReconnectWait }
func (c *Config) NATSDrainTimeout() time.Duration  { return c.NATS.DrainTimeout }
func (c *Config) NATSPingInterval() time.Duration  { return c.NATS.PingInterval }
func (c *Config) NATSMaxPingsOut() int             { return c.NATS.MaxPingsOut }
func (c *Config) NATSClientName() string           { return c.NATS.ClientName }
