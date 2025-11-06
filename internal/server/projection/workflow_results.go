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

package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	constant "github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// WorkflowResults updates the result KV bucket when workflows complete.
func WorkflowResults(ctx context.Context, conn *jetstreamx.Connection, conv serde.BinarySerde) error {
	consumer, err := conn.EnsureConsumer(ctx, constant.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       constant.WorkflowResultProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: constant.HistoryFilterSubjectPattern,
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow result projector consumer: %w", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event api.WorkflowEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			msg.Term()
			return
		}
		slog.Info("PROJECTOR/WF_RESULT: got history event", "event", event)

		switch e := event.(type) {
		case *api.WorkflowCompleted:
			result, _ := conv.SerializeBinary(e.Result)
			slog.Info(fmt.Sprintf("PROJECTOR/WF_RESULT: updating KV: %v - %v", e.ID, string(result)))
			if _, err := conn.Set(ctx, constant.WorkflowResultBucket, string(e.ID), result); err != nil {
				slog.Error("PROJECTOR/WF_RESULT: failed to set workflow result", "workflow_id", e.ID, "error", err)
				msg.Nak()
				return
			}
		}
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("workflow result projector failed: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	slog.Debug("Workflow result projector stopped.")
	return nil
}
