package projection

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	constant "github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// WorkflowTasks creates workflow tasks based on history events.
func WorkflowTasks(ctx context.Context, conn *jetstreamx.Connection, conv serde.BinarySerde) error {
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
		event, err := decodeWorkflowEvent(msg, conv)
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/WF: could not decode event, terminating: %v", err))
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
		var workflowID api.WorkflowID

		switch e := event.(type) {
		case *api.WorkflowStarted:
			slog.Info(fmt.Sprintf("PROJECTOR/WF: got WorkflowStartedEvent: %v", event))
			shouldCreateTask = true
			workflowFnName = e.WorkflowFnName
			inputArgs = e.Input
			workflowID = e.ID

		case *api.ActivityCompleted:
			shouldCreateTask = true
			workflowFnName = e.WorkflowFnName
			workflowID = e.ID
			inputArgs, err = loadWorkflowInputArgs(ctx, js, conv, workflowFnName, workflowID)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: %v", err))
				msg.Term()
				return
			}

		case *api.ActivityFailed:
			slog.Info(fmt.Sprintf("PROJECTOR/WF: got ActivityFailed event: %v", event))
			shouldCreateTask = true
			workflowFnName = e.WorkflowFnName
			workflowID = e.ID
			inputArgs, err = loadWorkflowInputArgs(ctx, js, conv, workflowFnName, workflowID)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: %v", err))
				msg.Term()
				return
			}
		case *api.ActivityScheduled:
			// Activity scheduling does not require a workflow task; it's handled by the activity projector.
			slog.Debug("PROJECTOR/WF: ignoring ActivityScheduled event for workflow task projection", "workflow_id", e.ID)
			msg.Ack()
			return
		default:
			slog.Debug("PROJECTOR/WF: ignoring history event", "type", fmt.Sprintf("%T", event))
			msg.Ack()
			return
		}

		if shouldCreateTask {
			idStr := workflowID.String()

			task := api.WorkflowTask{
				WorkflowID: idStr,
				WorkflowFn: workflowFnName,
				Input:      inputArgs,
			}

			// Use the provided BinarySerde instead of hardcoded JSON
			taskData, err := conv.SerializeBinary(task)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/WF: failed to serialize task: %v", err))
				msg.Nak()
				return
			}

			taskSubject := fmt.Sprintf("workflow.%s.tasks", idStr)
			_, err = js.PublishMsg(
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

func loadWorkflowInputArgs(ctx context.Context, js jetstream.JetStream, conv serde.BinarySerde, workflowFnName string, workflowID api.WorkflowID) ([]any, error) {
	kv, err := js.KeyValue(ctx, constant.WorkflowInputBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV store: %w", err)
	}

	key := api.WorkflowInputKey(workflowFnName, workflowID)
	kvEntry, err := kv.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get input args from KV for %s: %w", key, err)
	}

	var inputArgs []any
	if err := conv.DeserializeBinary(kvEntry.Value(), &inputArgs); err != nil {
		return nil, fmt.Errorf("failed to deserialize input args from KV: %w", err)
	}
	return inputArgs, nil
}
