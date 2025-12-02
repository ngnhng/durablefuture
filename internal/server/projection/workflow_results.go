package projection

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// WorkflowResults updates the result KV bucket when workflows complete.
func WorkflowResults(ctx context.Context, conn *jetstreamx.Connection, serder serde.BinarySerde) error {
	consumer, err := conn.EnsureConsumer(ctx, api.WorkflowHistoryStream, jetstream.ConsumerConfig{
		Durable:       api.WorkflowResultProjectorConsumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: api.HistoryFilterSubjectPattern,
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow result projector consumer: %w", err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		event, err := decodeWorkflowEvent(msg, serder)
		if err != nil {
			slog.Info(fmt.Sprintf("PROJECTOR/WF_RESULT: could not decode event, terminating: %v", err))
			msg.Term()
			return
		}
		slog.Info("PROJECTOR/WF_RESULT: got history event", "event", event)

		switch e := event.(type) {
		case *api.WorkflowCompleted:
			result, _ := serder.SerializeBinary(e.Result)
			slog.Info(fmt.Sprintf("PROJECTOR/WF_RESULT: updating KV: %v - %v", e.ID, string(result)))
			if _, err := conn.Set(ctx, api.WorkflowResultBucket, string(e.ID), result); err != nil {
				slog.ErrorContext(ctx, "PROJECTOR/WF_RESULT: failed to set workflow result", "workflow_id", e.ID, "error", err)
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
	slog.InfoContext(ctx, "Workflow result projector stopped.")
	return nil
}
