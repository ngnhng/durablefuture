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
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// ActivityRetries listens for ActivityRetried events and reschedules activity tasks after backoff delay.
func ActivityRetries(ctx context.Context, conn *jetstreamx.Connection, conv serde.BinarySerde) error {
	js, _ := conn.JS()

	consumer, err := conn.EnsureConsumer(ctx, api.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       api.ActivityRetryProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: api.HistoryFilterSubjectPattern,
	})
	if err != nil {
		return fmt.Errorf("failed to create activity retry projector consumer: %w", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		event, err := decodeWorkflowEvent(msg, conv)
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/RETRY: could not decode event, terminating: %v", err))
			msg.Term()
			return
		}

		_, err = msg.Metadata()
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/RETRY: could not get event metadata, terminating: %v", err))
			msg.Term()
			return
		}

		switch e := event.(type) {
		case *api.ActivityRetried:
			// Get the original ActivityScheduled event to retrieve full task context
			// For now, we'll need to reconstruct the task from what we know
			// In a production system, you might want to query the history stream for the original event

			slog.Info("PROJECTOR/RETRY: processing retry event",
				"workflow_id", e.ID,
				"activity_fn", e.ActivityFnName,
				"attempt", e.Attempt,
				"delay_ms", e.NextRetryDelay,
			)

			// Sleep for the backoff delay (blocking this goroutine)
			// In production, you might want to use NATS delayed delivery or a separate timer service
			retryDelay := time.Duration(e.NextRetryDelay) * time.Millisecond

			// Spawn a goroutine to handle the delayed retry so we don't block the consumer
			go func() {
				time.Sleep(retryDelay)

				// We need to get the full activity task context from the workflow history
				// For now, we'll fetch it from the workflow's KV store or reconstruct from history
				activityTask, err := fetchActivityTaskFromHistory(ctx, conn, conv, e)
				if err != nil {
					slog.Error("PROJECTOR/RETRY: failed to fetch activity task context", "error", err, "workflow_id", e.ID)
					return
				}

				// Increment attempt number
				activityTask.Attempt = e.Attempt + 1

				// Serialize and publish the retry task
				taskData, err := conv.SerializeBinary(activityTask)
				if err != nil {
					slog.Error("PROJECTOR/RETRY: failed to serialize retry task", "error", err)
					return
				}

				taskSubject := fmt.Sprintf("activity.%s.tasks", strings.Split(string(e.ID), ".")[0])
				msgID := fmt.Sprintf("actask-retry-%s-%d-%d", e.ID, e.Attempt+1, time.Now().UnixNano())

				if _, err = js.PublishMsg(
					ctx,
					&nats.Msg{
						Subject: taskSubject,
						Data:    taskData,
					},
					jetstream.WithMsgID(msgID),
				); err != nil {
					slog.Error("PROJECTOR/RETRY: failed to publish retry task", "error", err, "workflow_id", e.ID)
					return
				}

				slog.Info("PROJECTOR/RETRY: rescheduled activity task",
					"workflow_id", e.ID,
					"activity_fn", e.ActivityFnName,
					"attempt", activityTask.Attempt,
					"subject", taskSubject,
				)
			}()

		default:
			slog.Debug("PROJECTOR/RETRY: ignoring history event", "type", fmt.Sprintf("%T", event))
		}
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("activity retry projector failed: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	slog.Debug("Activity retry projector stopped.")
	return nil
}

// fetchActivityTaskFromHistory reconstructs the ActivityTask from workflow history
func fetchActivityTaskFromHistory(ctx context.Context, conn *jetstreamx.Connection, conv serde.BinarySerde, retryEvent *api.ActivityRetried) (*api.ActivityTask, error) {
	js, _ := conn.JS()

	// Get the workflow history stream
	historySubject := fmt.Sprintf("history.%s", retryEvent.ID)

	// Fetch all events for this workflow to find the ActivityScheduled event
	consumer, err := js.CreateOrUpdateConsumer(ctx, api.WorkflowHistoryStream, jetstream.ConsumerConfig{
		FilterSubject: historySubject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckNonePolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create history consumer: %w", err)
	}

	// Fetch messages to find the ActivityScheduled event
	var scheduledEvent *api.ActivityScheduled
	var scheduledTime time.Time

	msgs, err := consumer.Fetch(100) // Fetch up to 100 events
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	for msg := range msgs.Messages() {
		event, err := decodeWorkflowEvent(msg, conv)
		if err != nil {
			continue
		}

		// Look for the ActivityScheduled event matching this activity
		if scheduled, ok := event.(*api.ActivityScheduled); ok && scheduled.ActivityFnName == retryEvent.ActivityFnName {
			scheduledEvent = scheduled
			if meta, err := msg.Metadata(); err == nil {
				scheduledTime = meta.Timestamp
			}
			break
		}
	}

	if scheduledEvent == nil {
		return nil, fmt.Errorf("could not find ActivityScheduled event for %s in workflow %s", retryEvent.ActivityFnName, retryEvent.ID)
	}

	// Reconstruct the activity task with updated attempt
	return &api.ActivityTask{
		WorkflowFn:               retryEvent.WorkflowFnName,
		WorkflowID:               string(retryEvent.ID),
		ActivityFn:               retryEvent.ActivityFnName,
		Input:                    scheduledEvent.Input,
		Attempt:                  retryEvent.Attempt + 1,
		ScheduleToCloseTimeoutMs: scheduledEvent.ScheduleToCloseTimeoutMs,
		StartToCloseTimeoutMs:    scheduledEvent.StartToCloseTimeoutMs,
		RetryPolicy:              scheduledEvent.RetryPolicy,
		ScheduledAtMs:            scheduledTime.UnixMilli(),
	}, nil
}
