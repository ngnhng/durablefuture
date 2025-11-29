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

	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// DemoHandler handles demo workflow HTTP endpoints
type DemoHandler struct {
	conn  *jetstreamx.Connection
	serde serde.BinarySerde

	mu              sync.RWMutex
	workflowStates  map[string]*WorkflowState
	currentWorkflow string
}

// NewDemoHandler creates a new demo handler
func NewDemoHandler(conn *jetstreamx.Connection, serde serde.BinarySerde) *DemoHandler {
	h := &DemoHandler{
		conn:           conn,
		serde:          serde,
		workflowStates: make(map[string]*WorkflowState),
	}

	// Start watching workflow events in the background
	go h.watchWorkflowEvents(context.Background())

	return h
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
		slog.Error("failed to marshal attributes", "error", err)
		http.Error(w, fmt.Sprintf("Failed to marshal attributes: %v", err), http.StatusInternalServerError)
		return
	}
	cmd.Attributes = attrBytes

	data, err := h.serde.SerializeBinary(cmd)
	if err != nil {
		slog.Error("failed to marshal command", "error", err)
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
	slog.Debug("demo: received start reply", "size", len(msg.Data), "reply_subject", msg.Subject)

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

	slog.Info("demo: workflow started successfully", "workflow_id", workflowID, "order_id", orderID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"workflow_id": workflowID,
		"status":      "started",
	}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// GetStatus returns the current workflow status
func (h *DemoHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	currentID := h.currentWorkflow
	state := h.workflowStates[currentID]
	h.mu.RUnlock()

	if state == nil {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status": "no_workflow",
		}); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		slog.Error("failed to encode workflow state", "error", err)
	}
}

// Crash forcefully terminates the server for demo purposes
func (h *DemoHandler) Crash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "crashing",
	}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}

	slog.Warn("server crash requested via demo endpoint")

	// Give response time to send before exiting
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(1)
	}()
}
