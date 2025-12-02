package client

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidWorkflowFunction is returned when the provided workflow function is invalid
	ErrInvalidWorkflowFunction = errors.New("invalid workflow function")

	// ErrWorkflowNotFound is returned when a workflow cannot be found
	ErrWorkflowNotFound = errors.New("workflow not found")

	// ErrWorkflowAlreadyRunning is returned when attempting to start a workflow that is already running
	ErrWorkflowAlreadyRunning = errors.New("workflow already running")

	// ErrNoConnection is returned when the NATS connection is not established
	ErrNoConnection = errors.New("no NATS connection")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")
)

// WorkflowExecutionError represents an error that occurred during workflow execution
type WorkflowExecutionError struct {
	WorkflowID string
	Cause      error
	Message    string
}

func (e *WorkflowExecutionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("workflow %s failed: %s", e.WorkflowID, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("workflow %s failed: %v", e.WorkflowID, e.Cause)
	}
	return fmt.Sprintf("workflow %s failed", e.WorkflowID)
}

func (e *WorkflowExecutionError) Unwrap() error {
	return e.Cause
}

// NewWorkflowExecutionError creates a new WorkflowExecutionError
func NewWorkflowExecutionError(workflowID string, cause error) *WorkflowExecutionError {
	return &WorkflowExecutionError{
		WorkflowID: workflowID,
		Cause:      cause,
	}
}

// ActivityError represents an error that occurred during activity execution
type ActivityError struct {
	ActivityName string
	WorkflowID   string
	Attempt      int
	Cause        error
}

func (e *ActivityError) Error() string {
	return fmt.Sprintf("activity %s (workflow=%s, attempt=%d) failed: %v",
		e.ActivityName, e.WorkflowID, e.Attempt, e.Cause)
}

func (e *ActivityError) Unwrap() error {
	return e.Cause
}

// NewActivityError creates a new ActivityError
func NewActivityError(activityName, workflowID string, attempt int, cause error) *ActivityError {
	return &ActivityError{
		ActivityName: activityName,
		WorkflowID:   workflowID,
		Attempt:      attempt,
		Cause:        cause,
	}
}
