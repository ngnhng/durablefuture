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

package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/internal/server/config"
	"github.com/ngnhng/durablefuture/internal/server/logger"
)

type Options struct {
	NATSHost string
	NATSPort string
}

func Run(ctx context.Context, opts Options) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Allow CLI flags to override NATS host/port.
	if opts.NATSHost != "" {
		cfg.NATS.Host = opts.NATSHost
	}
	if opts.NATSPort != "" {
		cfg.NATS.Port = opts.NATSPort
	}
	cfg.NATS.URL = fmt.Sprintf("nats://%s:%s", cfg.NATS.Host, cfg.NATS.Port)

	logger, err := logger.NewLogger(ctx, cfg)
	if err != nil {
		return err
	}
	slog.SetDefault(logger.Slogger)
	defer func() {
		if logger.LoggerProvider != nil {
			if err := logger.LoggerProvider.Shutdown(ctx); err != nil {
				slog.Error("failed to shut down logger provider", "error", err)
			}
		}
	}()

	mgr, err := NewManager(ctx, cfg, &serde.JsonSerde{})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- mgr.Run(ctx)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		slog.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	slog.Info("manager shutting down")
	cancel()
	return nil
}
