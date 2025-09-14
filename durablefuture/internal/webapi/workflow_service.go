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

package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	workflowv1 "durablefuture/api/workflow/v1"
	"durablefuture/api/workflow/v1/workflowv1connect"
	"durablefuture/examples"
	"durablefuture/sdk/client"
	"durablefuture/workflow"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WorkflowServer implements the ConnectRPC WorkflowService
type WorkflowServer struct {
	workflowClient client.Client
	// In-memory storage for demonstration - in production this would be persisted
	workflows map[string]*workflowv1.WorkflowExecution
	events    map[string][]*workflowv1.WorkflowEvent
}

// NewWorkflowServer creates a new WorkflowServer instance
func NewWorkflowServer() (*WorkflowServer, error) {
	wfClient, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow client: %w", err)
	}

	return &WorkflowServer{
		workflowClient: wfClient,
		workflows:      make(map[string]*workflowv1.WorkflowExecution),
		events:         make(map[string][]*workflowv1.WorkflowEvent),
	}, nil
}

// GetWorkflows returns a list of all workflows
func (s *WorkflowServer) GetWorkflows(
	ctx context.Context,
	req *connect.Request[workflowv1.GetWorkflowsRequest],
) (*connect.Response[workflowv1.GetWorkflowsResponse], error) {
	slog.InfoContext(ctx, "GetWorkflows called")

	// Convert map to slice for response
	var workflows []*workflowv1.WorkflowExecution
	for _, wf := range s.workflows {
		workflows = append(workflows, wf)
	}

	// Simple pagination for demo
	pageSize := int(req.Msg.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	start := 0
	if req.Msg.PageToken != "" {
		if pageStart, err := strconv.Atoi(req.Msg.PageToken); err == nil {
			start = pageStart
		}
	}

	end := start + pageSize
	if end > len(workflows) {
		end = len(workflows)
	}

	var paginatedWorkflows []*workflowv1.WorkflowExecution
	if start < len(workflows) {
		paginatedWorkflows = workflows[start:end]
	}

	nextPageToken := ""
	if end < len(workflows) {
		nextPageToken = strconv.Itoa(end)
	}

	resp := &workflowv1.GetWorkflowsResponse{
		Workflows:     paginatedWorkflows,
		NextPageToken: nextPageToken,
	}

	return connect.NewResponse(resp), nil
}

// GetWorkflowHistory returns the execution history of a specific workflow
func (s *WorkflowServer) GetWorkflowHistory(
	ctx context.Context,
	req *connect.Request[workflowv1.GetWorkflowHistoryRequest],
) (*connect.Response[workflowv1.GetWorkflowHistoryResponse], error) {
	slog.InfoContext(ctx, "GetWorkflowHistory called", "workflow_id", req.Msg.WorkflowId)

	events := s.events[req.Msg.WorkflowId]
	if events == nil {
		events = []*workflowv1.WorkflowEvent{}
	}

	resp := &workflowv1.GetWorkflowHistoryResponse{
		Events: events,
	}

	return connect.NewResponse(resp), nil
}

// ExecuteWorkflow starts a new workflow execution
func (s *WorkflowServer) ExecuteWorkflow(
	ctx context.Context,
	req *connect.Request[workflowv1.ExecuteWorkflowRequest],
) (*connect.Response[workflowv1.ExecuteWorkflowResponse], error) {
	slog.InfoContext(ctx, "ExecuteWorkflow called", "workflow_name", req.Msg.WorkflowName)

	// Parse input JSON
	var input map[string]any
	if req.Msg.InputJson != "" {
		if err := json.Unmarshal([]byte(req.Msg.InputJson), &input); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid input JSON: %w", err))
		}
	}

	var workflowID string

	// Execute workflow based on name
	switch req.Msg.WorkflowName {
	case "OrderWorkflow":
		// Extract parameters for OrderWorkflow
		customerID, _ := input["customer_id"].(string)
		productID, _ := input["product_id"].(string)
		amount, _ := input["amount"].(float64)
		quantity := int(input["quantity"].(float64))

		future, execErr := s.workflowClient.ExecuteWorkflow(ctx, examples.OrderWorkflow,
			customerID, productID, amount, quantity)
		if execErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to execute workflow: %w", execErr))
		}

		// Cast to workflow.Future to access WorkflowID
		if execution, ok := future.(*workflow.Execution); ok {
			workflowID = execution.WorkflowID
		} else {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected future type"))
		}

		// Store workflow execution info
		s.workflows[workflowID] = &workflowv1.WorkflowExecution{
			WorkflowId:   workflowID,
			WorkflowName: req.Msg.WorkflowName,
			Status:       workflowv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING,
			StartTime:    timestamppb.Now(),
			Input:        req.Msg.InputJson,
		}

		// Store initial events
		s.events[workflowID] = []*workflowv1.WorkflowEvent{
			{
				EventId:     fmt.Sprintf("evt_%d", time.Now().UnixNano()),
				WorkflowId:  workflowID,
				EventType:   workflowv1.EventType_EVENT_TYPE_WORKFLOW_STARTED,
				Timestamp:   timestamppb.Now(),
				Details:     fmt.Sprintf("Workflow %s started", req.Msg.WorkflowName),
			},
		}

		// Start a goroutine to monitor workflow completion
		go s.monitorWorkflow(ctx, workflowID, future)

	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown workflow: %s", req.Msg.WorkflowName))
	}

	resp := &workflowv1.ExecuteWorkflowResponse{
		WorkflowId: workflowID,
		Status:     workflowv1.WorkflowStatus_WORKFLOW_STATUS_RUNNING,
	}

	return connect.NewResponse(resp), nil
}

// monitorWorkflow monitors workflow completion in the background
func (s *WorkflowServer) monitorWorkflow(ctx context.Context, workflowID string, future workflow.Future) {
	// Create a separate context for monitoring to avoid cancellation issues
	monitorCtx := context.Background()
	
	var result any
	err := future.Get(monitorCtx, &result)

	// Update workflow status and add completion event
	if workflow, exists := s.workflows[workflowID]; exists {
		if err != nil {
			workflow.Status = workflowv1.WorkflowStatus_WORKFLOW_STATUS_FAILED
			workflow.EndTime = timestamppb.Now()

			// Add failure event
			if events := s.events[workflowID]; events != nil {
				s.events[workflowID] = append(events, &workflowv1.WorkflowEvent{
					EventId:    fmt.Sprintf("evt_%d", time.Now().UnixNano()),
					WorkflowId: workflowID,
					EventType:  workflowv1.EventType_EVENT_TYPE_WORKFLOW_FAILED,
					Timestamp:  timestamppb.Now(),
					Details:    fmt.Sprintf("Workflow failed: %v", err),
				})
			}
		} else {
			workflow.Status = workflowv1.WorkflowStatus_WORKFLOW_STATUS_COMPLETED
			workflow.EndTime = timestamppb.Now()

			// Convert result to JSON
			if resultBytes, marshalErr := json.Marshal(result); marshalErr == nil {
				workflow.Output = string(resultBytes)
			}

			// Add completion event
			if events := s.events[workflowID]; events != nil {
				s.events[workflowID] = append(events, &workflowv1.WorkflowEvent{
					EventId:    fmt.Sprintf("evt_%d", time.Now().UnixNano()),
					WorkflowId: workflowID,
					EventType:  workflowv1.EventType_EVENT_TYPE_WORKFLOW_COMPLETED,
					Timestamp:  timestamppb.Now(),
					Details:    fmt.Sprintf("Workflow completed successfully"),
				})
			}
		}
	}
}

// Handler returns the ConnectRPC handler for the workflow service
func (s *WorkflowServer) Handler() (string, http.Handler) {
	return workflowv1connect.NewWorkflowServiceHandler(s)
}