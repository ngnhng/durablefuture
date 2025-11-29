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

package command

import (
	"testing"

	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
)

func TestNewHandler(t *testing.T) {
	// Test with nil values to ensure NewHandler doesn't panic
	handler := NewHandler(nil, nil)

	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}
}

func TestErrorReplySerialization(t *testing.T) {
	serde := &serde.MsgpackSerde{}

	tests := []struct {
		name       string
		workflowID string
		errorMsg   string
	}{
		{
			name:       "error with workflow ID",
			workflowID: "test-workflow-123",
			errorMsg:   "something went wrong",
		},
		{
			name:       "error without workflow ID",
			workflowID: "",
			errorMsg:   "failed to process",
		},
		{
			name:       "empty error message",
			workflowID: "test-workflow-456",
			errorMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error reply
			reply := api.StartWorkflowReply{
				WorkflowID: tt.workflowID,
				Error:      tt.errorMsg,
			}

			// Serialize
			data, err := serde.SerializeBinary(reply)
			if err != nil {
				t.Fatalf("SerializeBinary failed: %v", err)
			}

			// Deserialize
			var decoded api.StartWorkflowReply
			if err := serde.DeserializeBinary(data, &decoded); err != nil {
				t.Fatalf("DeserializeBinary failed: %v", err)
			}

			// Verify
			if decoded.WorkflowID != tt.workflowID {
				t.Errorf("WorkflowID = %v, want %v", decoded.WorkflowID, tt.workflowID)
			}

			if decoded.Error != tt.errorMsg {
				t.Errorf("Error = %v, want %v", decoded.Error, tt.errorMsg)
			}
		})
	}
}

func TestCommandSerialization(t *testing.T) {
	serde := &serde.MsgpackSerde{}

	tests := []struct {
		name        string
		commandType api.WorkflowCommandType
	}{
		{
			name:        "start workflow command",
			commandType: api.StartWorkflowCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command
			cmd := api.Command{
				CommandType: tt.commandType,
				Attributes:  []byte("test-attributes"),
			}

			// Serialize
			data, err := serde.SerializeBinary(cmd)
			if err != nil {
				t.Fatalf("SerializeBinary failed: %v", err)
			}

			// Deserialize
			var decoded api.Command
			if err := serde.DeserializeBinary(data, &decoded); err != nil {
				t.Fatalf("DeserializeBinary failed: %v", err)
			}

			// Verify
			if decoded.CommandType != tt.commandType {
				t.Errorf("CommandType = %v, want %v", decoded.CommandType, tt.commandType)
			}
		})
	}
}

func TestStartWorkflowAttributesSerialization(t *testing.T) {
	serde := &serde.MsgpackSerde{}

	tests := []struct {
		name           string
		workflowFnName string
		input          []any
	}{
		{
			name:           "simple input",
			workflowFnName: "TestWorkflow",
			input:          []any{"arg1", 123, true},
		},
		{
			name:           "empty input",
			workflowFnName: "EmptyWorkflow",
			input:          []any{},
		},
		{
			name:           "nil input",
			workflowFnName: "NilWorkflow",
			input:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create attributes
			attrs := api.StartWorkflowAttributes{
				WorkflowFnName: tt.workflowFnName,
				Input:          tt.input,
			}

			// Serialize
			data, err := serde.SerializeBinary(attrs)
			if err != nil {
				t.Fatalf("SerializeBinary failed: %v", err)
			}

			// Deserialize
			var decoded api.StartWorkflowAttributes
			if err := serde.DeserializeBinary(data, &decoded); err != nil {
				t.Fatalf("DeserializeBinary failed: %v", err)
			}

			// Verify
			if decoded.WorkflowFnName != tt.workflowFnName {
				t.Errorf("WorkflowFnName = %v, want %v", decoded.WorkflowFnName, tt.workflowFnName)
			}

			if len(decoded.Input) != len(tt.input) {
				t.Errorf("Input length = %v, want %v", len(decoded.Input), len(tt.input))
			}
		})
	}
}
