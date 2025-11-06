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
		return nil, err
	}

	if cfg.NATS.URL == "" {
		cfg.NATS.URL = fmt.Sprintf("nats://%s:%s", cfg.NATS.Host, cfg.NATS.Port)
	}

	return &cfg, nil
}

func (c *Config) ServiceName() string {
	return c.Service
}

func (c *Config) GetVersion() string {
	return c.Version
}
