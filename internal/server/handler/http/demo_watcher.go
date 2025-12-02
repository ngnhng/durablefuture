package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
)

// watchWorkflowEvents subscribes to workflow history and updates state
func (h *DemoHandler) watchWorkflowEvents(ctx context.Context) {
	// Wait for stream to be ready with exponential backoff
	time.Sleep(2 * time.Second) // Initial delay to let streams be created

	js, err := h.conn.JS()
	if err != nil {
		slog.Error("failed to get jetstream", "error", err)
		return
	}

	// Retry consumer creation with backoff
	var consumer jetstream.Consumer
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		consumer, err = js.CreateOrUpdateConsumer(ctx, "WORKFLOW_HISTORY", jetstream.ConsumerConfig{
			Durable:       "demo-ui-consumer",
			FilterSubject: "history.*",
			AckPolicy:     jetstream.AckExplicitPolicy,
			DeliverPolicy: jetstream.DeliverAllPolicy,
		})
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			backoff := time.Duration(i+1) * time.Second
			slog.Debug("stream not ready, retrying...", "attempt", i+1, "backoff", backoff)
			time.Sleep(backoff)
		}
	}

	if err != nil {
		slog.Error("failed to create demo consumer after retries", "error", err)
		return
	}

	slog.Info("demo UI consumer created successfully")

	msgs, err := consumer.Messages()
	if err != nil {
		slog.Error("failed to get messages", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := msgs.Next()
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				slog.Error("error receiving message", "error", err)
				continue
			}

			h.handleEvent(msg)
			msg.Ack()
		}
	}
}

// handleEvent processes workflow events and updates the workflow state
func (h *DemoHandler) handleEvent(msg jetstream.Msg) {
	// Get event type from header
	eventType := msg.Headers().Get(api.ChronicleEventNameHeader)
	if eventType == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Parse the event to get the workflow ID
	var baseEvent struct {
		ID api.WorkflowID `json:"id"`
	}
	if err := h.serde.DeserializeBinary(msg.Data(), &baseEvent); err != nil {
		slog.Error("failed to deserialize base event", "error", err)
		return
	}

	workflowID := baseEvent.ID.String()
	state, exists := h.workflowStates[workflowID]
	if !exists {
		return
	}

	switch eventType {
	case "workflow/started":
		state.Status = "running"
		state.StartedAt = time.Now()

	case "activity/scheduled":
		var evt api.ActivityScheduled
		if err := h.serde.DeserializeBinary(msg.Data(), &evt); err != nil {
			return
		}

		stepIdx := h.findOrCreateStep(state, evt.ActivityFnName)
		if stepIdx >= 0 {
			now := time.Now()
			state.StepDetails[stepIdx].Status = "running"
			state.StepDetails[stepIdx].Attempts++
			state.StepDetails[stepIdx].StartedAt = &now
			state.CurrentStep = stepIdx + 1
			if state.StepDetails[stepIdx].Attempts == 0 {
				state.StepDetails[stepIdx].Attempts = 1
			}
		}

	case "activity/completed":
		var evt api.ActivityCompleted
		if err := h.serde.DeserializeBinary(msg.Data(), &evt); err != nil {
			return
		}

		stepIdx := h.findStepByName(state, evt.ActivityFnName)
		if stepIdx >= 0 {
			now := time.Now()
			state.StepDetails[stepIdx].Status = "completed"
			state.StepDetails[stepIdx].EndedAt = &now
			if len(state.StepDetails[stepIdx].RetryHistory) > 0 {
				state.StepDetails[stepIdx].Attempts = len(state.StepDetails[stepIdx].RetryHistory) + 1
			}
		}

	case "activity/failed":
		var evt api.ActivityFailed
		if err := h.serde.DeserializeBinary(msg.Data(), &evt); err != nil {
			return
		}

		stepIdx := h.findStepByName(state, evt.ActivityFnName)
		if stepIdx >= 0 {
			state.StepDetails[stepIdx].Status = "failed"
			state.StepDetails[stepIdx].Error = evt.Error
			// Don't set EndedAt - it might retry
			if current := len(state.StepDetails[stepIdx].RetryHistory) + 1; current > state.StepDetails[stepIdx].Attempts {
				state.StepDetails[stepIdx].Attempts = current
			}
		}

	case "activity/retried":
		var evt api.ActivityRetried
		if err := h.serde.DeserializeBinary(msg.Data(), &evt); err != nil {
			return
		}

		stepIdx := h.findStepByName(state, evt.ActivityFnName)
		if stepIdx >= 0 {
			step := &state.StepDetails[stepIdx]
			step.Status = "running"
			step.Error = ""
			now := time.Now()
			step.RetryHistory = append(step.RetryHistory, RetryAttempt{
				Attempt:        int(evt.Attempt),
				Error:          evt.Error,
				NextRetryDelay: evt.NextRetryDelay,
				RecordedAt:     now,
			})
			if len(step.RetryHistory)+1 > step.Attempts {
				step.Attempts = len(step.RetryHistory) + 1
			}
		}

	case "workflow/completed":
		var evt api.WorkflowCompleted
		if err := h.serde.DeserializeBinary(msg.Data(), &evt); err != nil {
			return
		}

		state.Status = "completed"
		now := time.Now()
		state.CompletedAt = &now

		// Convert result to JSON for display
		if len(evt.Result) > 0 {
			resultBytes, _ := json.Marshal(evt.Result[0])
			state.Result = resultBytes
		}

	case "workflow/failed":
		var evt api.WorkflowFailed
		if err := h.serde.DeserializeBinary(msg.Data(), &evt); err != nil {
			return
		}

		state.Status = "failed"
		now := time.Now()
		state.CompletedAt = &now
		state.Error = evt.Error
	}
}
