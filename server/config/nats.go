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
