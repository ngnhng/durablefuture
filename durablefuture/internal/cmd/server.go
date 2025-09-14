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

package cmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"durablefuture/internal/logger"
	"durablefuture/internal/manager"
	"durablefuture/internal/types"
	"durablefuture/internal/webapi"
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

	logger, err := logger.NewLogger(ctx, &logger.LoggerOptions{
		Mode:   types.ModeDebug,
		Writer: os.Stdout,
	})

	slog.SetDefault(logger.Slogger)
	// TODO: setup otel
	// global.SetLoggerProvider(logger.LoggerProvider)

	defer func() {
		if logger.LoggerProvider != nil {
			if err := logger.LoggerProvider.Shutdown(ctx); err != nil {
				slog.Error("failed to shut down logger provider", "error", err)
			}
		}
	}()
	slog.Info("Logger initialized successfully")

	mgr, err := manager.NewManager(ctx, fmt.Sprintf("nats://%s:%s", host, port))
	if err != nil {
		slog.Error("Error creating manager: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 2)
	
	// Start the manager
	go func() {
		errCh <- mgr.Run(ctx)
	}()

	// Start the web API server
	go func() {
		webServer, webErr := sc.startWebServer(ctx)
		if webErr != nil {
			slog.ErrorContext(ctx, "Failed to start web server", "error", webErr)
			errCh <- webErr
			return
		}
		errCh <- webServer.Start(ctx)
	}()

	select {
	case <-sigCh:
		slog.InfoContext(ctx, "Shutdown signal received...")
	case err := <-errCh:
		if err != nil {
			slog.ErrorContext(ctx, "Server error", "error", err)
			return err
		}
	}

	slog.InfoContext(ctx, "Shutting down...")
	cancel()
	return nil
}

// startWebServer initializes and returns the web API server
func (sc *ServerCommand) startWebServer(ctx context.Context) (*webapi.Server, error) {
	webServer, err := webapi.NewServer(ctx, ":8080")
	if err != nil {
		return nil, fmt.Errorf("failed to create web server: %w", err)
	}
	
	slog.InfoContext(ctx, "Web server created successfully", "addr", ":8080")
	return webServer, nil
}
