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
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"durablefuture/internal/utils"

	"durablefuture/internal/types"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// runWorkflowTaskProjector creates workflow tasks based on history events.
func (m *Manager) runWorkflowTaskProjector(ctx context.Context) error {
	js, _ := m.conn.JS()

	// This consumer follows the HISTORY stream. It needs a durable name so it can resume.
	consumer, err := m.conn.EnsureConsumer(ctx, "HISTORY", jetstream.ConsumerConfig{
		Durable:       "projector-workflow-tasks",
		DeliverPolicy: jetstream.DeliverAllPolicy, // Process all history on startup
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "history.>",
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow task projector consumer: %w", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event types.WorkflowEvent
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

		// Decide if this event type should trigger a new workflow decision task.
		shouldCreateTask := false
		var workflowFnName string
		var inputArgs []any

		switch event.EventType {
		case types.WorkflowStartedEvent:
			slog.Info(fmt.Sprintf("workflow projector got WorkflowStartedEvent: %v", event))
			shouldCreateTask = true
			var attrs types.WorkflowStartedAttributes
			if err := json.Unmarshal(event.Attributes, &attrs); err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to unmarshal WorkflowStartedAttributes, terminating msg: %v", err))
				msg.Term()
				return
			}
			workflowFnName = attrs.WorkflowFnName
			inputArgs = attrs.Input

		case types.ActivityCompletedEvent:
			slog.Info(fmt.Sprintf("workflow projector got ActivityCompletedEvent: %v", string(event.Attributes)))
			var attrs types.ActivityCompletedAttributes
			if err := json.Unmarshal(event.Attributes, &attrs); err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to unmarshal ActivityCompletedAttributes, terminating msg: %v", err))
				msg.Term()
				return
			}
			shouldCreateTask = true
			workflowFnName = attrs.WorkflowFnName

			// Get input args from KV store using workflow function name
			kv, err := js.KeyValue(ctx, "workflow-input")
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

		case types.ActivityFailedEvent:
			slog.Info(fmt.Sprintf("workflow projector got: %v", event))

			shouldCreateTask = true
			// For these events, we might need the workflow function name too.
			// This implies it should be stored somewhere accessible, like on the task,
			// or the WorkflowTask message might not need it if the worker can look it up.
			// Let's assume for now the worker can look it up by workflowID if needed.
			workflowFnName = ""
		}

		if shouldCreateTask {
			workflowID := strings.Split(msg.Subject(), ".")[1]

			task := types.WorkflowTask{
				WorkflowID: workflowID,
				WorkflowFn: workflowFnName,
				Input:      inputArgs,
			}
			taskData, _ := json.Marshal(task)

			// Use the event sequence as a message ID for deduplication.
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
			slog.Info(fmt.Sprintf("PROJECTOR/WF: Created workflow task for %s triggered by event %v", workflowID, event.EventType))
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

// runActivityTaskProjector creates activity tasks based on history events.
func (m *Manager) runActivityTaskProjector(ctx context.Context) error {
	js, _ := m.conn.JS()

	consumer, err := m.conn.EnsureConsumer(ctx, "HISTORY", jetstream.ConsumerConfig{
		Durable:       "projector-activity-tasks",
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "history.>",
	})
	if err != nil {
		slog.Info(fmt.Sprintf("error: %v", err))
		return err
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event types.WorkflowEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			slog.Info(fmt.Sprintf("error: %v", err))

			msg.Term()
			return
		}

		meta, err := msg.Metadata()
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/ACT: could not get event metadata, terminating: %v", err))
			msg.Term()
			return
		}

		slog.Debug(fmt.Sprintf("ACTIVITY_PROJECTOR_DEBUG: Received event - EventType=%d, WorkflowID=%s", event.EventType, event.WorkflowID))

		if event.EventType == types.ActivityTaskScheduledEvent {
			slog.Debug(fmt.Sprintf("✅ activity projector got: %v", string(msg.Data())))

			var attrs types.ActivityTaskScheduledAttributes
			if err := json.Unmarshal(event.Attributes, &attrs); err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/ACT: failed to unmarshal ActivityTaskScheduledAttributes, terminating msg: %v", err))
				msg.Term()
				return
			}

			workflowID := event.WorkflowID

			slog.Info(fmt.Sprintf("projector/act: prepping workflowID: %s", workflowID))

			task := types.ActivityTask{
				WorkflowFn: attrs.WorkflowFnName,
				WorkflowID: workflowID,
				ActivityFn: attrs.ActivityFnName,
				Input:      attrs.Input,
			}
			slog.Info(fmt.Sprintf("projector/act: prepping payload: %v", task))

			taskData, err := json.Marshal(task)
			if err != nil {
				slog.Info(fmt.Sprintf("error: %v", err))
				return
			}

			taskSubject := fmt.Sprintf("activity.%s.tasks", workflowID)
			slog.Info(fmt.Sprintf("projector/act: prepping task subject: %s", taskSubject))

			_, err = js.Publish(
				ctx,
				taskSubject,
				taskData,
				jetstream.WithMsgID(fmt.Sprintf("actask-%s-%d", workflowID, meta.Sequence.Consumer)))

			if err != nil {
				slog.Debug(fmt.Sprintf("❌ PROJECTOR/ACT: failed to publish activity task for %s: %v", workflowID, err))
				msg.Nak()
				return
			}
			slog.Debug(fmt.Sprintf("✅ PROJECTOR/ACT: Published/Created activity task '%s' for workflow %s", attrs.ActivityFnName, workflowID))
		} else {
			slog.Debug(fmt.Sprintf("❌ ACTIVITY_PROJECTOR_DEBUG: Ignoring event type %d (not ActivityTaskScheduledEvent=%d)", event.EventType, types.ActivityTaskScheduledEvent))
		}

		msg.Ack()
		slog.Debug(fmt.Sprintf("✅ ACTIVITY_PROJECTOR_DEBUG: Acknowledged message for event type %d", event.EventType))

	})
	if err != nil {
		slog.Debug(fmt.Sprintf("error: %v", err))

		return err
	}

	<-ctx.Done()
	cc.Stop()
	slog.Debug("Activity task projector stopped.")
	return nil
}

func (m *Manager) runWorkflowResultProjector(ctx context.Context) error {
	consumer, err := m.conn.EnsureConsumer(ctx, "HISTORY", jetstream.ConsumerConfig{
		Durable:       "projector-workflow-result",
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "history.>",
	})
	if err != nil {
		return err
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {

		var event types.WorkflowEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			msg.Term()
			return
		}
		slog.Info(fmt.Sprintf("WF_RESULT_PROJECTOR: got history event: %s", utils.DebugWorkflowEvents([]types.WorkflowEvent{event})))

		if event.EventType == types.WorkflowCompleted {
			slog.Info(fmt.Sprintf("WF_RESULT_PROJECTOR: processing WorkflowCompleted: %v", event))
			var attrs types.WorkflowCompletedAttributes
			if err := json.Unmarshal(event.Attributes, &attrs); err != nil {
				slog.Info(fmt.Sprintf("WF_RESULT_PROJECTOR: failed to unmarshal WorkflowCompletedAttributes, terminating msg: %v", err))
				msg.Term()
				return
			}

			workflowID := event.WorkflowID
			result, _ := json.Marshal(attrs.Result)
			slog.Info(fmt.Sprintf("WF_RESULT_PROJECTOR: updating KV: %v - %v", workflowID, string(result)))
			m.conn.Set(ctx, "workflow-result", workflowID, result)

		}
		msg.Ack()
	})
	if err != nil {
		return err
	}

	<-ctx.Done()
	cc.Stop()
	slog.Debug("Activity task projector stopped.")
	return nil

}
