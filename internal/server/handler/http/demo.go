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

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

type DemoHandler struct {
	conn  *jetstreamx.Connection
	serde serde.BinarySerde

	mu              sync.RWMutex
	workflowStates  map[string]*WorkflowState
	currentWorkflow string
}

type WorkflowState struct {
	WorkflowID  string          `json:"workflow_id"`
	Status      string          `json:"status"` // "pending", "running", "completed", "failed"
	CurrentStep int             `json:"current_step"`
	Steps       []string        `json:"steps"`
	StepDetails []StepDetail    `json:"step_details"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

type StepDetail struct {
	Name         string         `json:"name"`
	Status       string         `json:"status"` // "pending", "running", "completed", "failed"
	Attempts     int            `json:"attempts"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	EndedAt      *time.Time     `json:"ended_at,omitempty"`
	Error        string         `json:"error,omitempty"`
	RetryHistory []RetryAttempt `json:"retry_history,omitempty"`
}

type RetryAttempt struct {
	Attempt        int       `json:"attempt"`
	Error          string    `json:"error,omitempty"`
	NextRetryDelay int64     `json:"next_retry_delay_ms,omitempty"`
	RecordedAt     time.Time `json:"recorded_at"`
}

func NewDemoHandler(conn *jetstreamx.Connection, serde serde.BinarySerde) *DemoHandler {
	h := &DemoHandler{
		conn:           conn,
		serde:          serde,
		workflowStates: make(map[string]*WorkflowState),
	}

	// Start watching workflow events
	go h.watchWorkflowEvents(context.Background())

	return h
}

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

	slog.Info("Demo UI consumer created successfully")

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

func (h *DemoHandler) handleEvent(msg jetstream.Msg) {
	// Get event type from header
	eventType := msg.Headers().Get(api.ChronicleEventNameHeader)
	if eventType == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// We need to parse the event to get the workflow ID
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

		// Find or create step detail
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

func (h *DemoHandler) findOrCreateStep(state *WorkflowState, activityName string) int {
	for i, step := range state.StepDetails {
		if step.Name == activityName {
			return i
		}
	}

	// Create new step
	state.StepDetails = append(state.StepDetails, StepDetail{
		Name:   activityName,
		Status: "pending",
	})
	return len(state.StepDetails) - 1
}

func (h *DemoHandler) findStepByName(state *WorkflowState, activityName string) int {
	for i, step := range state.StepDetails {
		if step.Name == activityName {
			return i
		}
	}
	return -1
}

// StartWorkflow starts a new demo workflow
func (h *DemoHandler) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate order ID for the workflow input
	orderID := fmt.Sprintf("demo-%d", time.Now().UnixNano())
	slog.Info("demo: request to start workflow", "order_id", orderID)

	// Publish workflow start command using Request-Reply to get the actual workflow ID
	cmdSubject := "command.request.start"
	cmd := api.Command{
		CommandType: api.StartWorkflowCommand,
		Attributes:  nil, // Will be set below
	}

	attrs := api.StartWorkflowAttributes{
		WorkflowFnName: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.OrderWithRetriesWorkflow",
		Input: []any{
			orderID,
			"demo-customer",
			299.99,
		},
	}

	attrBytes, err := h.serde.SerializeBinary(attrs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal attributes: %v", err), http.StatusInternalServerError)
		return
	}
	cmd.Attributes = attrBytes

	data, err := h.serde.SerializeBinary(cmd)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal command: %v", err), http.StatusInternalServerError)
		return
	}
	slog.Debug("demo: serialized start command", "bytes", len(data))

	// Use Request to get the reply with the actual workflow ID
	msg, err := h.conn.NATS().Request(cmdSubject, data, 10*time.Second)
	if err != nil {
		slog.Error("demo: failed to publish start command", "subject", cmdSubject, "error", err)
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}
	slog.Debug("demo: received start reply", "size", len(msg.Data), "reply_subject", msg.Subject, "payload", string(msg.Data))

	// Deserialize the reply to get the workflow ID
	var reply api.StartWorkflowReply
	if err := h.serde.DeserializeBinary(msg.Data, &reply); err != nil {
		slog.Error("demo: failed to deserialize start reply", "error", err)
		http.Error(w, fmt.Sprintf("Failed to parse reply: %v", err), http.StatusInternalServerError)
		return
	}

	if reply.Error != "" {
		slog.Error("demo: start reply contained error", "workflow_id", reply.WorkflowID, "error", reply.Error)
		http.Error(w, fmt.Sprintf("Server error: %s", reply.Error), http.StatusInternalServerError)
		return
	}

	workflowID := reply.WorkflowID

	// Initialize workflow state with the server-assigned workflow ID
	h.mu.Lock()
	h.workflowStates[workflowID] = &WorkflowState{
		WorkflowID:  workflowID,
		Status:      "pending",
		CurrentStep: 0,
		Steps: []string{
			"github.com/ngnhng/durablefuture/examples/scenarios/order-retries.ValidateOrderDetailsActivity",
			"github.com/ngnhng/durablefuture/examples/scenarios/order-retries.TransientFailurePaymentActivity",
			"github.com/ngnhng/durablefuture/examples/scenarios/order-retries.ReserveInventoryActivity",
			"github.com/ngnhng/durablefuture/examples/scenarios/order-retries.PackageOrderActivity",
			"github.com/ngnhng/durablefuture/examples/scenarios/order-retries.EventuallySuccessfulShippingActivity",
			"github.com/ngnhng/durablefuture/examples/scenarios/order-retries.NotifyCustomerActivity",
		},
		StepDetails: []StepDetail{
			{Name: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.ValidateOrderDetailsActivity", Status: "pending"},
			{Name: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.TransientFailurePaymentActivity", Status: "pending"},
			{Name: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.ReserveInventoryActivity", Status: "pending"},
			{Name: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.PackageOrderActivity", Status: "pending"},
			{Name: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.EventuallySuccessfulShippingActivity", Status: "pending"},
			{Name: "github.com/ngnhng/durablefuture/examples/scenarios/order-retries.NotifyCustomerActivity", Status: "pending"},
		},
		StartedAt: time.Now(),
	}
	h.currentWorkflow = workflowID
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"workflow_id": workflowID,
		"status":      "started",
	})
}

// GetStatus returns the current workflow status
func (h *DemoHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	currentID := h.currentWorkflow
	state := h.workflowStates[currentID]
	h.mu.RUnlock()

	if state == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "no_workflow",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// Crash forcefully terminates the server
func (h *DemoHandler) Crash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "crashing",
	})

	slog.Warn("Server crash requested via demo endpoint")

	// Give response time to send
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(1)
	}()
}
