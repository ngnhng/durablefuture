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
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// ActivityTasks creates activity tasks based on workflow history events.
func ActivityTasks(ctx context.Context, conn *jetstreamx.Connection, _ serde.BinarySerde) error {
	js, _ := conn.JS()

	consumer, err := conn.EnsureConsumer(ctx, api.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       api.ActivityTaskProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: api.HistoryFilterSubjectPattern,
	})
	if err != nil {
		return fmt.Errorf("failed to create activity task projector consumer: %w", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event api.WorkflowEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/ACT: could not unmarshal event, terminating: %v", err))
			msg.Term()
			return
		}

		meta, err := msg.Metadata()
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/ACT: could not get event metadata, terminating: %v", err))
			msg.Term()
			return
		}

		switch e := event.(type) {
		case *api.ActivityScheduled:
			task := api.ActivityTask{
				WorkflowFn: e.WorkflowFnName,
				WorkflowID: string(e.ID),
				ActivityFn: e.ActivityFnName,
				Input:      e.Input,
			}

			taskData, err := json.Marshal(task)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/ACT: failed to marshal task payload: %v", err))
				msg.Term()
				return
			}

			taskSubject := fmt.Sprintf("activity.%s.tasks", e.ID)
			if _, err = js.Publish(
				ctx,
				taskSubject,
				taskData,
				jetstream.WithMsgID(fmt.Sprintf("actask-%s-%d", e.ID, meta.Sequence.Consumer)),
			); err != nil {
				slog.Debug(fmt.Sprintf("PROJECTOR/ACT: failed to publish activity task for %s: %v", e.ID, err))
				msg.Nak()
				return
			}
		}
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("activity task projector failed: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	slog.Debug("Activity task projector stopped.")
	return nil
}
