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

import (
	"fmt"
	"testing"
	"time"
)

func TestRetryPolicy_CalculateRetryDelay(t *testing.T) {
	tests := []struct {
		name           string
		policy         *RetryPolicy
		attempt        int
		expectedDelay  time.Duration
	}{
		{
			name: "exponential backoff first attempt",
			policy: &RetryPolicy{
				InitialInterval:    1 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    60 * time.Second,
				BackoffPolicy:      BackoffExponential,
			},
			attempt:       0,
			expectedDelay: 1 * time.Second,
		},
		{
			name: "exponential backoff second attempt",
			policy: &RetryPolicy{
				InitialInterval:    1 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    60 * time.Second,
				BackoffPolicy:      BackoffExponential,
			},
			attempt:       1,
			expectedDelay: 2 * time.Second,
		},
		{
			name: "exponential backoff with max interval",
			policy: &RetryPolicy{
				InitialInterval:    1 * time.Second,
				BackoffCoefficient: 2.0,
				MaximumInterval:    5 * time.Second,
				BackoffPolicy:      BackoffExponential,
			},
			attempt:       3, // Would be 8 seconds without max
			expectedDelay: 5 * time.Second,
		},
		{
			name: "linear backoff",
			policy: &RetryPolicy{
				InitialInterval: 2 * time.Second,
				MaximumInterval: 60 * time.Second,
				BackoffPolicy:   BackoffLinear,
			},
			attempt:       2, // Should be 2 * (2+1) = 6 seconds
			expectedDelay: 6 * time.Second,
		},
		{
			name: "fixed backoff",
			policy: &RetryPolicy{
				InitialInterval: 3 * time.Second,
				MaximumInterval: 60 * time.Second,
				BackoffPolicy:   BackoffFixed,
			},
			attempt:       5,
			expectedDelay: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := tt.policy.CalculateRetryDelay(tt.attempt)
			if delay != tt.expectedDelay {
				t.Errorf("CalculateRetryDelay() = %v, want %v", delay, tt.expectedDelay)
			}
		})
	}
}

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	tests := []struct {
		name           string
		policy         *RetryPolicy
		attempt        int
		err            error
		expectedResult bool
	}{
		{
			name: "within retry limit",
			policy: &RetryPolicy{
				MaximumAttempts: 3,
			},
			attempt:        1,
			err:            fmt.Errorf("temporary error"),
			expectedResult: true,
		},
		{
			name: "exceeded retry limit",
			policy: &RetryPolicy{
				MaximumAttempts: 3,
			},
			attempt:        3,
			err:            fmt.Errorf("temporary error"),
			expectedResult: false,
		},
		{
			name: "unlimited retries",
			policy: &RetryPolicy{
				MaximumAttempts: -1,
			},
			attempt:        10,
			err:            fmt.Errorf("temporary error"),
			expectedResult: true,
		},
		{
			name: "non-retryable error",
			policy: &RetryPolicy{
				MaximumAttempts:        3,
				NonRetryableErrorTypes: []string{"fatal error", "validation error"},
			},
			attempt:        1,
			err:            fmt.Errorf("fatal error"),
			expectedResult: false,
		},
		{
			name: "retryable error not in non-retryable list",
			policy: &RetryPolicy{
				MaximumAttempts:        3,
				NonRetryableErrorTypes: []string{"fatal error"},
			},
			attempt:        1,
			err:            fmt.Errorf("temporary error"),
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.policy.ShouldRetry(tt.attempt, tt.err)
			if result != tt.expectedResult {
				t.Errorf("ShouldRetry() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()
	
	if policy.MaximumAttempts != 3 {
		t.Errorf("Expected MaximumAttempts = 3, got %d", policy.MaximumAttempts)
	}
	
	if policy.InitialInterval != 1*time.Second {
		t.Errorf("Expected InitialInterval = 1s, got %v", policy.InitialInterval)
	}
	
	if policy.BackoffCoefficient != 2.0 {
		t.Errorf("Expected BackoffCoefficient = 2.0, got %f", policy.BackoffCoefficient)
	}
	
	if policy.BackoffPolicy != BackoffExponential {
		t.Errorf("Expected BackoffPolicy = BackoffExponential, got %v", policy.BackoffPolicy)
	}
}

func TestDefaultActivityOptions(t *testing.T) {
	options := DefaultActivityOptions()
	
	if options.TaskQueue != "default" {
		t.Errorf("Expected TaskQueue = 'default', got '%s'", options.TaskQueue)
	}
	
	if options.RetryPolicy == nil {
		t.Error("Expected RetryPolicy to be set")
	}
	
	if options.ScheduleToCloseTimeout != 5*time.Minute {
		t.Errorf("Expected ScheduleToCloseTimeout = 5m, got %v", options.ScheduleToCloseTimeout)
	}
}

func TestDefaultWorkflowOptions(t *testing.T) {
	options := DefaultWorkflowOptions()
	
	if options.TaskQueue != "default" {
		t.Errorf("Expected TaskQueue = 'default', got '%s'", options.TaskQueue)
	}
	
	if options.Namespace != "default" {
		t.Errorf("Expected Namespace = 'default', got '%s'", options.Namespace)
	}
	
	if options.WorkflowExecutionTimeout != 10*time.Minute {
		t.Errorf("Expected WorkflowExecutionTimeout = 10m, got %v", options.WorkflowExecutionTimeout)
	}
}