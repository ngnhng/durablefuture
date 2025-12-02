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
	HTTPPort string
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

	// Use MessagePack for better performance and type preservation
	httpPort := opts.HTTPPort
	if httpPort == "" {
		httpPort = "8080"
	}

	mgr, err := NewManager(ctx, cfg, &serde.MsgpackSerde{}, httpPort)
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
