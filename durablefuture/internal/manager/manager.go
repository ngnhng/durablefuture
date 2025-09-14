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

package manager

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"durablefuture/internal/constants"
	"durablefuture/internal/converter"
	natz "durablefuture/internal/natz"

	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	conn    *natz.Conn
	handler *Handler
}

func NewDefaultManager(ctx context.Context) (*Manager, error) {
	conn, err := natz.Connect(natz.DefaultConfig())
	if err != nil {
		return nil, err
	}

	m := &Manager{
		conn: conn,
		handler: &Handler{
			conn: conn,
			conv: converter.NewJsonConverter()},
	}

	m.ensureStreams(ctx)
	m.ensureKV(ctx)

	return m, nil
}

func NewManager(ctx context.Context, natsURL string) (*Manager, error) {
	config := natz.DefaultConfig()
	config.URL = natsURL

	conn, err := natz.Connect(config)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		conn: conn,
		handler: &Handler{
			conn: conn,
			conv: converter.NewJsonConverter()},
	}

	m.ensureStreams(ctx)
	m.ensureKV(ctx)

	return m, nil
}

func (m *Manager) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting Command Consumer...")
		return m.runCommandRequestHandler(gCtx)
	})

	g.Go(func() error {
		slog.Info("Starting Workflow Task Projector...")
		return m.runWorkflowTaskProjector(gCtx)
	})

	g.Go(func() error {
		slog.Info("Starting Activity Task Projector...")
		return m.runActivityTaskProjector(gCtx)
	})

	g.Go(func() error {
		slog.Info("Starting Workflow Task Projector...")
		return m.runWorkflowResultProjector(gCtx)
	})

	slog.Info("Manager is running.")
	return g.Wait()
}

func (m *Manager) ensureStreams(ctx context.Context) error {
	var err error
	// Workflow History Stream
	_, err = m.conn.EnsureStream(ctx, jetstream.StreamConfig{
		Name:      constants.HistoryStream,
		Subjects:  []string{constants.HistorySubjectPattern},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		log.Printf("error ensuring HISTORY stream: %v", err)
		return fmt.Errorf("failed to ensure workflow history stream: %w", err)
	}

	// Workflow Tasks Stream
	_, err = m.conn.EnsureStream(ctx, jetstream.StreamConfig{
		Name:      constants.WorkflowTasksStream,
		Subjects:  []string{constants.WorkflowTasksSubjectPattern},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		log.Printf("error ensuring WF_TASKS stream: %v", err)

		return fmt.Errorf("failed to ensure workflow tasks stream: %w", err)
	}

	// Activity Tasks Stream, read-only so no listen subject
	_, err = m.conn.EnsureStream(ctx, jetstream.StreamConfig{
		Name:      constants.ActivityTasksStream,
		Subjects:  []string{constants.ActivityTasksSubjectPattern},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		log.Printf("error ensuring AC_TASKS stream: %v", err)

		return fmt.Errorf("failed to ensure activity tasks stream: %w", err)
	}
	return nil
}

func (m *Manager) ensureKV(ctx context.Context) error {
	var err error
	_, err = m.conn.EnsureKV(ctx, jetstream.KeyValueConfig{
		Bucket: constants.WorkflowResultBucket,
	})

	if err != nil {
		return err
	}

	_, err = m.conn.EnsureKV(ctx, jetstream.KeyValueConfig{
		Bucket: constants.WorkflowInputBucket,
	})

	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) runCommandRequestHandler(ctx context.Context) error {
	sub, err := m.conn.QueueSubscribe(
		constants.CommandRequestSubjectPattern,
		constants.ManagerCommandProcessorsConsumer,
		m.handler.HandleRequest,
	)
	if err != nil {
		return err
	}
	<-ctx.Done()
	sub.Unsubscribe()
	return nil
}
