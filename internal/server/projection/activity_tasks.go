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

// ActivityTasks creates activity tasks based on workflow history events.
func ActivityTasks(ctx context.Context, conn *jetstreamx.Connection, conv serde.BinarySerde) error {
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
		event, err := decodeWorkflowEvent(msg, conv)
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/ACT: could not decode event, terminating: %v", err))
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
				WorkflowFn:                 e.WorkflowFnName,
				WorkflowID:                 string(e.ID),
				ActivityFn:                 e.ActivityFnName,
				Input:                      e.Input,
				Attempt:                    1, // First attempt
				ScheduleToCloseTimeoutUnix: e.ScheduleToCloseTimeoutUnix,
				StartToCloseTimeoutUnix:    e.StartToCloseTimeoutUnix,
				RetryPolicy:                e.RetryPolicy,
				ScheduledAtMs:              time.Now().UnixMilli(),
			}

			// Use the provided BinarySerde instead of hardcoded JSON
			taskData, err := conv.SerializeBinary(task)
			if err != nil {
				slog.Info(fmt.Sprintf("PROJECTOR/ACT: failed to serialize task payload: %v", err))
				msg.Term()
				return
			}

			taskSubject := fmt.Sprintf("activity.%s.tasks", strings.Split(msg.Subject(), ".")[1])
			msgID := fmt.Sprintf("actask-%s-%d", e.ID, meta.Sequence.Consumer)
			if _, err = js.PublishMsg(
				ctx,
				&nats.Msg{
					Subject: taskSubject,
					Data:    taskData,
				},
				jetstream.WithMsgID(msgID),
			); err != nil {
				slog.Debug(fmt.Sprintf("PROJECTOR/ACT: failed to publish activity task for %s: %v", e.ID, err))
				msg.Nak()
				return
			}
			slog.Info("PROJECTOR/ACT: created activity task",
				"workflow_id", e.ID,
				"activity_fn", e.ActivityFnName,
				"subject", taskSubject,
				"msg_id", msgID,
				"attempt", 1,
			)
		default:
			slog.Debug("PROJECTOR/ACT: ignoring history event", "type", fmt.Sprintf("%T", event))
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
