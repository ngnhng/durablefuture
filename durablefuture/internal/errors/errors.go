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

package errors

import (
	"errors"
	"fmt"
)

// Common error types for the DurableFuture system

var (
	// ErrWorkflowExecution indicates an error during workflow execution
	ErrWorkflowExecution = errors.New("workflow execution error")
	
	// ErrActivityExecution indicates an error during activity execution
	ErrActivityExecution = errors.New("activity execution error")
	
	// ErrInvalidResult indicates an invalid result format
	ErrInvalidResult = errors.New("invalid result format")
	
	// ErrConfiguration indicates a configuration error
	ErrConfiguration = errors.New("configuration error")
	
	// ErrConnection indicates a connection error
	ErrConnection = errors.New("connection error")
)

// WorkflowExecutionError represents a workflow execution error with additional context
type WorkflowExecutionError struct {
	WorkflowID string
	Message    string
	Cause      error
}

func (e *WorkflowExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("workflow %s failed: %s: %v", e.WorkflowID, e.Message, e.Cause)
	}
	return fmt.Sprintf("workflow %s failed: %s", e.WorkflowID, e.Message)
}

func (e *WorkflowExecutionError) Unwrap() error {
	return e.Cause
}

func (e *WorkflowExecutionError) Is(target error) bool {
	return errors.Is(target, ErrWorkflowExecution)
}

// NewWorkflowExecutionError creates a new workflow execution error
func NewWorkflowExecutionError(workflowID, message string, cause error) *WorkflowExecutionError {
	return &WorkflowExecutionError{
		WorkflowID: workflowID,
		Message:    message,
		Cause:      cause,
	}
}

// ActivityExecutionError represents an activity execution error with additional context
type ActivityExecutionError struct {
	ActivityName string
	Message      string
	Cause        error
}

func (e *ActivityExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("activity %s failed: %s: %v", e.ActivityName, e.Message, e.Cause)
	}
	return fmt.Sprintf("activity %s failed: %s", e.ActivityName, e.Message)
}

func (e *ActivityExecutionError) Unwrap() error {
	return e.Cause
}

func (e *ActivityExecutionError) Is(target error) bool {
	return errors.Is(target, ErrActivityExecution)
}

// NewActivityExecutionError creates a new activity execution error
func NewActivityExecutionError(activityName, message string, cause error) *ActivityExecutionError {
	return &ActivityExecutionError{
		ActivityName: activityName,
		Message:      message,
		Cause:        cause,
	}
}