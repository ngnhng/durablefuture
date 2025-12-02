package worker

import (
	"errors"
	"fmt"
)

var (
	// ErrWorkflowNotRegistered is returned when a workflow is not registered with the worker
	ErrWorkflowNotRegistered = errors.New("workflow not registered")

	// ErrActivityNotRegistered is returned when an activity is not registered with the worker
	ErrActivityNotRegistered = errors.New("activity not registered")

	// ErrInvalidFunction is returned when attempting to register an invalid function
	ErrInvalidFunction = errors.New("invalid function: must be a function type")

	// ErrDuplicateRegistration is returned when attempting to register a function that is already registered
	ErrDuplicateRegistration = errors.New("function already registered")

	// ErrWorkerShutdown is returned when the worker is shutting down
	ErrWorkerShutdown = errors.New("worker is shutting down")
)

// RegistrationError represents an error that occurred during function registration
type RegistrationError struct {
	FunctionName string
	Cause        error
}

func (e *RegistrationError) Error() string {
	return fmt.Sprintf("failed to register function %s: %v", e.FunctionName, e.Cause)
}

func (e *RegistrationError) Unwrap() error {
	return e.Cause
}

// NewRegistrationError creates a new RegistrationError
func NewRegistrationError(functionName string, cause error) *RegistrationError {
	return &RegistrationError{
		FunctionName: functionName,
		Cause:        cause,
	}
}

// TaskProcessingError represents an error that occurred while processing a task
type TaskProcessingError struct {
	TaskType   string // "workflow" or "activity"
	TaskID     string
	WorkflowID string
	Cause      error
}

func (e *TaskProcessingError) Error() string {
	return fmt.Sprintf("failed to process %s task %s (workflow=%s): %v",
		e.TaskType, e.TaskID, e.WorkflowID, e.Cause)
}

func (e *TaskProcessingError) Unwrap() error {
	return e.Cause
}

// NewTaskProcessingError creates a new TaskProcessingError
func NewTaskProcessingError(taskType, taskID, workflowID string, cause error) *TaskProcessingError {
	return &TaskProcessingError{
		TaskType:   taskType,
		TaskID:     taskID,
		WorkflowID: workflowID,
		Cause:      cause,
	}
}
