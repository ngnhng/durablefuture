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
		return nil, err
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

	m.ensureStreams(ctx)
	m.ensureKV(ctx)

	return m, nil
}

func (m *Manager) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting HTTP Server...", "port", m.httpPort)
		return m.httpServer.Start(gCtx)
	})

	g.Go(func() error {
		slog.Info("Starting Command Consumer...")
		return command.RunProcessor(gCtx, m.conn, m.handler)
	})

	g.Go(func() error {
		slog.Info("Starting Workflow Task Projector...")
		return projection.WorkflowTasks(gCtx, m.conn, m.serde)
	})

	g.Go(func() error {
		slog.Info("Starting Activity Task Projector...")
		return projection.ActivityTasks(gCtx, m.conn, m.serde)
	})

	g.Go(func() error {
		slog.Info("Starting Activity Retry Projector...")
		return projection.ActivityRetries(gCtx, m.conn, m.serde)
	})

	g.Go(func() error {
		slog.Info("Starting Workflow Result Projector...")
		return projection.WorkflowResults(gCtx, m.conn, m.serde)
	})

	slog.Info("Manager is running.")
	return g.Wait()
}
