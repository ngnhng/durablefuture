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

package manager

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
	"github.com/ngnhng/durablefuture/server/utils"
)

// runWorkflowTaskProjector creates workflow tasks based on history events.
func (m *Manager) runWorkflowTaskProjector(ctx context.Context) error {
	js, _ := m.conn.JS()

	// This consumer follows the HISTORY stream. It needs a durable name so it can resume.
	consumer, err := m.conn.EnsureConsumer(ctx, constant.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       constant.WorkflowTaskProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy, // Process all history on startup
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

		// Decide if this event type should trigger a new workflow decision task.
		shouldCreateTask := false
		var workflowFnName string
		var inputArgs []any

		switch e := event.(type) {
		case *api.WorkflowStarted:
			slog.Info(fmt.Sprintf("workflow projector got WorkflowStartedEvent: %v", event))
			shouldCreateTask = true

			workflowFnName = e.WorkflowFnName
			inputArgs = e.Input

		case *api.ActivityCompleted:
			shouldCreateTask = true
			workflowFnName = e.WorkflowFnName

			// Get input args from KV store using workflow function name
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
			slog.Info(fmt.Sprintf("workflow projector got: %v", event))

			shouldCreateTask = true
			// For these events, we might need the workflow function name too.
			// This implies it should be stored somewhere accessible, like on the task,
			// or the WorkflowTask message might not need it if the worker can look it up.
			// Let's assume for now the worker can look it up by workflowID if needed.
			workflowFnName = ""
		default:
			panic("unhandled default case")
		}

		if shouldCreateTask {
			workflowID := strings.Split(msg.Subject(), ".")[1]

			task := api.WorkflowTask{
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

// runActivityTaskProjector creates activity tasks based on history events.
func (m *Manager) runActivityTaskProjector(ctx context.Context) error {
	js, _ := m.conn.JS()

	consumer, err := m.conn.EnsureConsumer(ctx, constant.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       constant.ActivityTaskProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: constant.HistoryFilterSubjectPattern,
	})
	if err != nil {
		slog.Info(fmt.Sprintf("error: %v", err))
		return err
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event api.WorkflowEvent
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

		switch e := event.(type) {
		case *api.ActivityScheduled:
			slog.Debug(fmt.Sprintf("✅ activity projector got: %v", string(msg.Data())))
			workflowId := e.ID
			slog.Info(fmt.Sprintf("projector/act: prepping workflowID: %s", workflowId))

			task := api.ActivityTask{
				WorkflowFn: e.WorkflowFnName,
				WorkflowID: string(workflowId),
				ActivityFn: e.ActivityFnName,
				Input:      e.Input,
			}
			slog.Info(fmt.Sprintf("projector/act: prepping payload: %v", task))

			taskData, err := json.Marshal(task)
			if err != nil {
				slog.Info(fmt.Sprintf("error: %v", err))
				return
			}

			taskSubject := fmt.Sprintf("activity.%s.tasks", workflowId)
			slog.Info(fmt.Sprintf("projector/act: prepping task subject: %s", taskSubject))

			_, err = js.Publish(
				ctx,
				taskSubject,
				taskData,
				jetstream.WithMsgID(fmt.Sprintf("actask-%s-%d", workflowId, meta.Sequence.Consumer)))
			if err != nil {
				slog.Debug(fmt.Sprintf("❌ PROJECTOR/ACT: failed to publish activity task for %s: %v", workflowId, err))
				msg.Nak()
				return
			}
		}
		msg.Ack()
		slog.Debug(fmt.Sprintf("✅ ACTIVITY_PROJECTOR_DEBUG: Acknowledged message for event %v", event))
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
	consumer, err := m.conn.EnsureConsumer(ctx, constant.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       constant.WorkflowResultProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: constant.HistoryFilterSubjectPattern,
	})
	if err != nil {
		return err
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {

		var event api.WorkflowEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			msg.Term()
			return
		}
		slog.Info(fmt.Sprintf("WF_RESULT_PROJECTOR: got history event: %s", utils.DebugWorkflowEvents([]api.WorkflowEvent{event})))

		switch e := event.(type) {
		case *api.WorkflowCompleted:
			result, _ := m.serde.SerializeBinary(e.Result)
			slog.Info(fmt.Sprintf("WF_RESULT_PROJECTOR: updating KV: %v - %v", e.ID, string(result)))
			m.conn.Set(ctx, constant.WorkflowResultBucket, string(e.ID), result)
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
