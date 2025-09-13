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

package types

import "time"

// BackoffPolicy defines how retry attempts are spaced out
type BackoffPolicy int

const (
	// BackoffFixed - fixed delay between retry attempts
	BackoffFixed BackoffPolicy = iota
	// BackoffLinear - linearly increasing delay (attempt * initial_delay)
	BackoffLinear
	// BackoffExponential - exponentially increasing delay (initial_delay * 2^attempt)
	BackoffExponential
)

// RetryPolicy defines how activities should be retried on failure
type RetryPolicy struct {
	// MaximumAttempts is the maximum number of retry attempts (0 = no retries, -1 = unlimited)
	MaximumAttempts int `json:"maximum_attempts"`
	
	// InitialInterval is the delay before the first retry attempt
	InitialInterval time.Duration `json:"initial_interval"`
	
	// BackoffCoefficient is the multiplier for exponential backoff (default: 2.0)
	BackoffCoefficient float64 `json:"backoff_coefficient"`
	
	// MaximumInterval is the maximum delay between retry attempts
	MaximumInterval time.Duration `json:"maximum_interval"`
	
	// BackoffPolicy determines how delays are calculated
	BackoffPolicy BackoffPolicy `json:"backoff_policy"`
	
	// NonRetryableErrorTypes are error messages/types that should not be retried
	NonRetryableErrorTypes []string `json:"non_retryable_error_types"`
}

// ActivityOptions defines execution options for activities
type ActivityOptions struct {
	// TaskQueue is the queue where the activity task will be dispatched
	TaskQueue string `json:"task_queue"`
	
	// ScheduleToCloseTimeout is the total time allowed for the activity
	ScheduleToCloseTimeout time.Duration `json:"schedule_to_close_timeout"`
	
	// StartToCloseTimeout is the time allowed for execution once started
	StartToCloseTimeout time.Duration `json:"start_to_close_timeout"`
	
	// ScheduleToStartTimeout is the time allowed to wait for a worker
	ScheduleToStartTimeout time.Duration `json:"schedule_to_start_timeout"`
	
	// HeartbeatTimeout is the heartbeat interval for long-running activities
	HeartbeatTimeout time.Duration `json:"heartbeat_timeout"`
	
	// RetryPolicy defines how the activity should be retried on failure
	RetryPolicy *RetryPolicy `json:"retry_policy"`
}

// WorkflowOptions defines execution options for workflows
type WorkflowOptions struct {
	// TaskQueue is the queue where the workflow task will be dispatched
	TaskQueue string `json:"task_queue"`
	
	// WorkflowExecutionTimeout is the total time allowed for the workflow
	WorkflowExecutionTimeout time.Duration `json:"workflow_execution_timeout"`
	
	// WorkflowRunTimeout is the time allowed for a single workflow run
	WorkflowRunTimeout time.Duration `json:"workflow_run_timeout"`
	
	// WorkflowTaskTimeout is the time allowed for processing a single workflow task
	WorkflowTaskTimeout time.Duration `json:"workflow_task_timeout"`
	
	// Namespace for multi-tenant isolation
	Namespace string `json:"namespace"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaximumAttempts:        3,
		InitialInterval:        1 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        60 * time.Second,
		BackoffPolicy:          BackoffExponential,
		NonRetryableErrorTypes: []string{},
	}
}

// DefaultActivityOptions returns sensible default activity options
func DefaultActivityOptions() *ActivityOptions {
	return &ActivityOptions{
		TaskQueue:              "default",
		ScheduleToCloseTimeout: 5 * time.Minute,
		StartToCloseTimeout:    1 * time.Minute,
		ScheduleToStartTimeout: 1 * time.Minute,
		HeartbeatTimeout:       30 * time.Second,
		RetryPolicy:            DefaultRetryPolicy(),
	}
}

// DefaultWorkflowOptions returns sensible default workflow options
func DefaultWorkflowOptions() *WorkflowOptions {
	return &WorkflowOptions{
		TaskQueue:                "default",
		WorkflowExecutionTimeout: 10 * time.Minute,
		WorkflowRunTimeout:       5 * time.Minute,
		WorkflowTaskTimeout:      10 * time.Second,
		Namespace:                "default",
	}
}

// CalculateRetryDelay calculates the delay for the next retry attempt
func (rp *RetryPolicy) CalculateRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rp.InitialInterval
	}
	
	var delay time.Duration
	switch rp.BackoffPolicy {
	case BackoffFixed:
		delay = rp.InitialInterval
	case BackoffLinear:
		delay = time.Duration(float64(rp.InitialInterval) * float64(attempt+1))
	case BackoffExponential:
		multiplier := 1.0
		for i := 0; i < attempt; i++ {
			multiplier *= rp.BackoffCoefficient
		}
		delay = time.Duration(float64(rp.InitialInterval) * multiplier)
	default:
		delay = rp.InitialInterval
	}
	
	if delay > rp.MaximumInterval {
		delay = rp.MaximumInterval
	}
	
	return delay
}

// ShouldRetry determines if an error should trigger a retry
func (rp *RetryPolicy) ShouldRetry(attempt int, err error) bool {
	// Check if we've exceeded maximum attempts
	if rp.MaximumAttempts >= 0 && attempt >= rp.MaximumAttempts {
		return false
	}
	
	// Check if this error type should not be retried
	errStr := err.Error()
	for _, nonRetryableErr := range rp.NonRetryableErrorTypes {
		if errStr == nonRetryableErr {
			return false
		}
	}
	
	return true
}