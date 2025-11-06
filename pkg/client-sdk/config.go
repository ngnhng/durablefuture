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

	"github.com/caarlos0/env/v11"
	internalconfig "github.com/ngnhng/durablefuture/internal/server/config"
)

// Config is the public SDK configuration users can construct or load from env.
type Config struct {
	NATS     internalconfig.NATSConfig    `json:"nats" envPrefix:"NATS_"`
	Timeouts internalconfig.TimeoutConfig `json:"timeouts" envPrefix:"TIMEOUTS_"`
}

// Load loads configuration from environment variables applying defaults.
func Load() (*Config, error) {
	cfg := Config{
		NATS: internalconfig.NATSConfig{
			Host:          internalconfig.DefaultNATSHost,
			Port:          internalconfig.DefaultNATSPort,
			MaxReconnects: internalconfig.DefaultMaxReconnects,
			ReconnectWait: internalconfig.DefaultReconnectWait,
			DrainTimeout:  internalconfig.DefaultDrainTimeout,
			PingInterval:  internalconfig.DefaultPingInterval,
			MaxPingsOut:   internalconfig.DefaultMaxPingsOut,
			ClientName:    "durablefuture-sdk",
		},
		Timeouts: internalconfig.TimeoutConfig{
			RequestTimeout: internalconfig.DefaultRequestTimeout,
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
