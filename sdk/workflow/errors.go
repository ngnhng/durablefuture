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

package workflow

import (
	"errors"
	"fmt"
)

var (
	// ErrActivityNotRegistered is returned when an activity is not registered with the worker
	ErrActivityNotRegistered = errors.New("activity not registered")

	// ErrInvalidActivityFunction is returned when the provided activity function is invalid
	ErrInvalidActivityFunction = errors.New("invalid activity function")

	// ErrNonDeterministicBehavior is returned when non-deterministic behavior is detected during replay
	ErrNonDeterministicBehavior = errors.New("non-deterministic behavior detected")

	// ErrPanicInWorkflow is returned when a workflow panics
	ErrPanicInWorkflow = errors.New("panic in workflow")
)

// NonRetryableError wraps an error to indicate it should not be retried
type NonRetryableError struct {
	Cause error
}

func (e *NonRetryableError) Error() string {
	return fmt.Sprintf("non-retryable error: %v", e.Cause)
}

func (e *NonRetryableError) Unwrap() error {
	return e.Cause
}

// NewNonRetryableError creates a new NonRetryableError
func NewNonRetryableError(err error) *NonRetryableError {
	return &NonRetryableError{Cause: err}
}

// IsNonRetryable checks if an error is non-retryable
func IsNonRetryable(err error) bool {
	var nre *NonRetryableError
	return errors.As(err, &nre)
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	TimeoutType string // "StartToClose", "ScheduleToClose", etc.
	Duration    string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("%s timeout after %s", e.TimeoutType, e.Duration)
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(timeoutType, duration string) *TimeoutError {
	return &TimeoutError{
		TimeoutType: timeoutType,
		Duration:    duration,
	}
}

// PanicError represents a panic that occurred in workflow code
type PanicError struct {
	Value interface{}
	Stack string
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("workflow panic: %v\nStack: %s", e.Value, e.Stack)
}

// NewPanicError creates a new PanicError
func NewPanicError(value interface{}, stack string) *PanicError {
	return &PanicError{
		Value: value,
		Stack: stack,
	}
}
