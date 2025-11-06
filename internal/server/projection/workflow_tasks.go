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
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	constant "github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// WorkflowTasks creates workflow tasks based on history events.
func WorkflowTasks(ctx context.Context, conn *jetstreamx.Connection, _ serde.BinarySerde) error {
	js, _ := conn.JS()

	consumer, err := conn.EnsureConsumer(ctx, constant.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       constant.WorkflowTaskProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: constant.HistoryFilterSubjectPattern,
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow task projector consumer: %w", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event api.WorkflowEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/WF: could not unmarshal event, terminating: %v", err))
			msg.Term()
			return
		}

		meta, err := msg.Metadata()
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/WF: could not get event metadata, terminating: %v", err))
			msg.Term()
			return
		}

		shouldCreateTask := false
		var workflowFnName string
		var inputArgs []any

		switch e := event.(type) {
		case *api.WorkflowStarted:
			slog.Info(fmt.Sprintf("PROJECTOR/WF: got WorkflowStartedEvent: %v", event))
			shouldCreateTask = true
			workflowFnName = e.WorkflowFnName
			inputArgs = e.Input

		case *api.ActivityCompleted:
			shouldCreateTask = true
			workflowFnName = e.WorkflowFnName

			kv, err := js.KeyValue(ctx, constant.WorkflowInputBucket)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to get KV store, terminating msg: %v", err))
				msg.Term()
				return
			}

			kvEntry, err := kv.Get(ctx, workflowFnName)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to get input args from KV for %s, terminating msg: %v", workflowFnName, err))
				msg.Term()
				return
			}

			if err := json.Unmarshal(kvEntry.Value(), &inputArgs); err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to unmarshal input args from KV, terminating msg: %v", err))
				msg.Term()
				return
			}

		case *api.ActivityFailed:
			slog.Info(fmt.Sprintf("PROJECTOR/WF: got ActivityFailed event: %v", event))
			shouldCreateTask = true
			workflowFnName = ""
		default:
			slog.Error("PROJECTOR/WF: unhandled event type", "type", fmt.Sprintf("%T", event))
		}

		if shouldCreateTask {
			workflowID := strings.Split(msg.Subject(), ".")[1]

			task := api.WorkflowTask{
				WorkflowID: workflowID,
				WorkflowFn: workflowFnName,
				Input:      inputArgs,
			}
			taskData, _ := json.Marshal(task)

			taskSubject := fmt.Sprintf("workflow.%s.tasks", workflowID)
			_, err := js.PublishMsg(
				ctx,
				&nats.Msg{
					Subject: taskSubject,
					Data:    taskData,
				},
				jetstream.WithMsgID(fmt.Sprintf("wftask-%s-%d", workflowID, meta.Sequence.Consumer)),
			)

			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to publish workflow task for %s: %v", workflowID, err))
				msg.Nak()
				return
			}
			slog.Info(fmt.Sprintf("PROJECTOR/WF: Created workflow task for %s triggered by event %v", workflowID, event))
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("workflow task projector failed: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	slog.Debug("Workflow task projector stopped.")
	return nil
}
