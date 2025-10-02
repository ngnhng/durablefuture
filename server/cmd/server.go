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

package cmd

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/server/config"
	"github.com/ngnhng/durablefuture/server/logger"
	"github.com/ngnhng/durablefuture/server/manager"
	"github.com/ngnhng/durablefuture/server/types"
	"go.opentelemetry.io/otel/log/global"
)

// ServerCommand implements the Command interface for starting the server
type ServerCommand struct {
	ctx context.Context
}

func NewServerCommand(ctx context.Context) *ServerCommand {
	return &ServerCommand{ctx: ctx}
}

// Execute implements Command.
// Runs the manager handling workflow commands and events
func (sc *ServerCommand) Execute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	port := fs.String("port", "4222", "NATS server port")
	host := fs.String("host", "localhost", "NATS server host")

	if err := fs.Parse(args); err != nil {
		return err
	}

	return sc.run(ctx, *host, *port)
}

// run contains the main server logic
func (sc *ServerCommand) run(ctx context.Context, host, port string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Initialize logger based on loaded configuration.
	lg, err := logger.NewLogger(ctx, cfg)
	if err != nil {
		return err
	}
	slog.SetDefault(lg.Slogger)
	if cfg.Mode == types.ModeRelease {
		global.SetLoggerProvider(lg.LoggerProvider)
	}
	defer func() {
		if lg.LoggerProvider != nil {
			if err := lg.LoggerProvider.Shutdown(ctx); err != nil {
				slog.Error("failed to shut down logger provider", "error", err)
			}
		}
	}()
	slog.Info("Logger initialized successfully")

	mgr, err := manager.NewManager(ctx, cfg, &serde.JsonSerde{})
	if err != nil {
		slog.Error("error creating manager", "error", err)
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- mgr.Run(ctx)
	}()

	select {
	case <-sigCh:
		slog.InfoContext(ctx, "Shutdown signal received...")
	case err := <-errCh:
		if err != nil {
			slog.ErrorContext(ctx, "manager error", "error", err)
			return err
		}
	}

	slog.InfoContext(ctx, "Shutting down...")
	cancel()
	return nil
}
