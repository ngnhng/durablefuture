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
	"strings"
	"testing"
	"time"

	"github.com/ngnhng/durablefuture/internal/server/types"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				Mode:    types.ModeDebug,
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
					PingInterval:  2 * time.Minute,
					MaxPingsOut:   2,
					ClientName:    "test-client",
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			config: &Config{
				Service: "",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "missing version",
			config: &Config{
				Service: "test-service",
				Version: "",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "missing NATS host",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "NATS host is required",
		},
		{
			name: "missing NATS port",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "NATS port is required",
		},
		{
			name: "invalid NATS port",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "invalid",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "invalid NATS port",
		},
		{
			name: "missing NATS URL",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "NATS URL is required",
		},
		{
			name: "invalid NATS max reconnects",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: -2,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "NATS max reconnects must be >= -1",
		},
		{
			name: "invalid NATS reconnect wait",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 0,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "NATS reconnect wait must be positive",
		},
		{
			name: "invalid NATS drain timeout",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  0,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "NATS drain timeout must be positive",
		},
		{
			name: "missing server host",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "",
					Port: "8080",
				},
			},
			wantErr: true,
			errMsg:  "server host is required",
		},
		{
			name: "missing server port",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "",
				},
			},
			wantErr: true,
			errMsg:  "server port is required",
		},
		{
			name: "invalid server port",
			config: &Config{
				Service: "test-service",
				Version: "v1.0.0",
				NATS: NATSConfig{
					Host:          "localhost",
					Port:          "4222",
					URL:           "nats://localhost:4222",
					MaxReconnects: 10,
					ReconnectWait: 2 * time.Second,
					DrainTimeout:  30 * time.Second,
				},
				Server: ServerConfig{
					Host: "localhost",
					Port: "not-a-number",
				},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConfig_ServiceName(t *testing.T) {
	cfg := &Config{
		Service: "test-service",
	}

	if got := cfg.ServiceName(); got != "test-service" {
		t.Errorf("ServiceName() = %v, want %v", got, "test-service")
	}
}

func TestConfig_GetVersion(t *testing.T) {
	cfg := &Config{
		Version: "v1.2.3",
	}

	if got := cfg.GetVersion(); got != "v1.2.3" {
		t.Errorf("GetVersion() = %v, want %v", got, "v1.2.3")
	}
}
