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

	"github.com/nats-io/nats.go/jetstream"
	constant "github.com/ngnhng/durablefuture/api"
)

func (m *Manager) ensureStreams(ctx context.Context) error {
	// Workflow History Stream
	_, err := m.conn.EnsureStream(ctx, jetstream.StreamConfig{
		Name:      constant.WorkflowHistoryStream,
		Subjects:  []string{constant.HistoryFilterSubjectPattern},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		slog.Error("failed to ensure workflow history stream", "error", err, "stream", constant.WorkflowHistoryStream)
		return fmt.Errorf("failed to ensure workflow history stream: %w", err)
	}
	slog.Info("ensured stream", "name", constant.WorkflowHistoryStream)

	// Workflow Tasks Stream
	_, err = m.conn.EnsureStream(ctx, jetstream.StreamConfig{
		Name:      constant.WorkflowTasksStream,
		Subjects:  []string{constant.WorkflowTasksFilterSubjectPattern},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		slog.Error("failed to ensure workflow tasks stream", "error", err, "stream", constant.WorkflowTasksStream)
		return fmt.Errorf("failed to ensure workflow tasks stream: %w", err)
	}
	slog.Info("ensured stream", "name", constant.WorkflowTasksStream)

	// Activity Tasks Stream
	_, err = m.conn.EnsureStream(ctx, jetstream.StreamConfig{
		Name:      constant.ActivityTasksStream,
		Subjects:  []string{constant.ActivityTasksFilterSubjectPattern},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		slog.Error("failed to ensure activity tasks stream", "error", err, "stream", constant.ActivityTasksStream)
		return fmt.Errorf("failed to ensure activity tasks stream: %w", err)
	}
	slog.Info("ensured stream", "name", constant.ActivityTasksStream)

	return nil
}

func (m *Manager) ensureKV(ctx context.Context) error {
	_, err := m.conn.EnsureKV(ctx, jetstream.KeyValueConfig{
		Bucket: constant.WorkflowResultBucket,
	})
	if err != nil {
		slog.Error("failed to ensure workflow result KV bucket", "error", err, "bucket", constant.WorkflowResultBucket)
		return fmt.Errorf("failed to ensure workflow result KV bucket: %w", err)
	}
	slog.Info("ensured KV bucket", "name", constant.WorkflowResultBucket)

	_, err = m.conn.EnsureKV(ctx, jetstream.KeyValueConfig{
		Bucket: constant.WorkflowInputBucket,
	})
	if err != nil {
		slog.Error("failed to ensure workflow input KV bucket", "error", err, "bucket", constant.WorkflowInputBucket)
		return fmt.Errorf("failed to ensure workflow input KV bucket: %w", err)
	}
	slog.Info("ensured KV bucket", "name", constant.WorkflowInputBucket)

	return nil
}
