package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/internal/server/handler/command"
	httphandler "github.com/ngnhng/durablefuture/internal/server/handler/http"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
	"github.com/ngnhng/durablefuture/internal/server/projection"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	conn       *jetstreamx.Connection
	handler    *command.Handler
	httpServer *httphandler.Server
	serde      serde.BinarySerde
	httpPort   string
}

func NewManager(ctx context.Context, cfg jetstreamx.Config, serde serde.BinarySerde, httpPort string) (*Manager, error) {
	conn, err := jetstreamx.Connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	if !conn.IsConnected() {
		return nil, fmt.Errorf("cannot connect to NATS instance")
	}

	m := &Manager{
		conn:       conn,
		handler:    command.NewHandler(conn, serde),
		httpServer: httphandler.NewServer(conn, serde, httpPort),
		serde:      serde,
		httpPort:   httpPort,
	}

	if err := m.ensureStreams(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ensure NATS streams: %w", err)
	}

	if err := m.ensureKV(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ensure NATS KV buckets: %w", err)
	}

	return m, nil
}

func (m *Manager) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("starting HTTP server", "port", m.httpPort)
		return m.httpServer.Start(gCtx)
	})

	g.Go(func() error {
		slog.Info("starting command processor")
		return command.RunProcessor(gCtx, m.conn, m.handler)
	})

	g.Go(func() error {
		slog.Info("starting workflow task projector")
		return projection.WorkflowTasks(gCtx, m.conn, m.serde)
	})

	g.Go(func() error {
		slog.Info("starting activity task projector")
		return projection.ActivityTasks(gCtx, m.conn, m.serde)
	})

	g.Go(func() error {
		slog.Info("starting activity retry projector")
		return projection.ActivityRetries(gCtx, m.conn, m.serde)
	})

	g.Go(func() error {
		slog.Info("starting workflow result projector")
		return projection.WorkflowResults(gCtx, m.conn, m.serde)
	})

	slog.Info("manager is running", "components", 6)

	// Wait for all goroutines to complete or context cancellation
	err := g.Wait()

	// Perform graceful shutdown
	slog.Info("initiating graceful shutdown")
	m.Shutdown()

	if err != nil && err != context.Canceled {
		slog.Error("manager stopped with error", "error", err)
		return err
	}

	slog.Info("manager shutdown complete")
	return nil
}

// Shutdown performs graceful shutdown of all manager components
func (m *Manager) Shutdown() {
	slog.Info("shutting down manager components")

	// Close NATS connection - this will drain and close all subscriptions
	if m.conn != nil {
		slog.Info("closing NATS connection")
		m.conn.Close()
		slog.Info("NATS connection closed")
	}
}
