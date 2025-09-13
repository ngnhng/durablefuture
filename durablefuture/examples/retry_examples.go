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

package examples

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"durablefuture/internal/types"
	"durablefuture/workflow"
)

// RetryExampleWorkflow demonstrates different retry strategies
func RetryExampleWorkflow(ctx workflow.Context, input string) (any, error) {
	log.Println("retry example workflow started")

	// Example 1: Activity with exponential backoff retry policy
	exponentialRetryOptions := &types.ActivityOptions{
		TaskQueue:              "default",
		ScheduleToCloseTimeout: 5 * time.Minute,
		StartToCloseTimeout:    30 * time.Second,
		RetryPolicy: &types.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			BackoffPolicy:      types.BackoffExponential,
		},
	}

	var exponentialResult string
	err := workflow.ExecuteActivityWithOptions(ctx, FlakyActivity, exponentialRetryOptions, input, 0.7).
		Get(ctx, &exponentialResult)
	if err != nil {
		log.Printf("Exponential backoff activity failed: %v", err)
		// Continue with other examples even if this fails
	}

	// Example 2: Activity with linear backoff retry policy
	linearRetryOptions := &types.ActivityOptions{
		TaskQueue:              "default", 
		ScheduleToCloseTimeout: 5 * time.Minute,
		StartToCloseTimeout:    30 * time.Second,
		RetryPolicy: &types.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: 2 * time.Second,
			MaximumInterval: 10 * time.Second,
			BackoffPolicy:   types.BackoffLinear,
		},
	}

	var linearResult string
	err = workflow.ExecuteActivityWithOptions(ctx, FlakyActivity, linearRetryOptions, input, 0.5).
		Get(ctx, &linearResult)
	if err != nil {
		log.Printf("Linear backoff activity failed: %v", err)
	}

	// Example 3: Activity with fixed delay retry policy
	fixedRetryOptions := &types.ActivityOptions{
		TaskQueue:              "default",
		ScheduleToCloseTimeout: 3 * time.Minute,
		StartToCloseTimeout:    20 * time.Second,
		RetryPolicy: &types.RetryPolicy{
			MaximumAttempts: 4,
			InitialInterval: 3 * time.Second,
			BackoffPolicy:   types.BackoffFixed,
		},
	}

	var fixedResult string
	err = workflow.ExecuteActivityWithOptions(ctx, FlakyActivity, fixedRetryOptions, input, 0.3).
		Get(ctx, &fixedResult)
	if err != nil {
		log.Printf("Fixed delay activity failed: %v", err)
	}

	// Example 4: Activity with non-retryable errors
	nonRetryableOptions := &types.ActivityOptions{
		TaskQueue:              "default",
		ScheduleToCloseTimeout: 2 * time.Minute,
		StartToCloseTimeout:    15 * time.Second,
		RetryPolicy: &types.RetryPolicy{
			MaximumAttempts:        3,
			InitialInterval:        1 * time.Second,
			BackoffPolicy:          types.BackoffExponential,
			NonRetryableErrorTypes: []string{"fatal error", "validation error"},
		},
	}

	var nonRetryResult string
	err = workflow.ExecuteActivityWithOptions(ctx, ActivityWithNonRetryableError, nonRetryableOptions, input).
		Get(ctx, &nonRetryResult)
	if err != nil {
		log.Printf("Non-retryable activity failed as expected: %v", err)
	}

	result := map[string]any{
		"exponential_result": exponentialResult,
		"linear_result":      linearResult,
		"fixed_result":       fixedResult,
		"non_retry_result":   nonRetryResult,
		"workflow_status":    "completed",
	}

	return result, nil
}

// FlakyActivity simulates an activity that fails randomly based on a failure rate
func FlakyActivity(ctx context.Context, input string, failureRate float64) (any, error) {
	log.Printf("FlakyActivity executing with input: %s, failure rate: %.2f", input, failureRate)

	// Simulate some processing time
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	// Random failure based on failure rate
	if rand.Float64() < failureRate {
		return nil, fmt.Errorf("random failure occurred (failure rate: %.2f)", failureRate)
	}

	result := fmt.Sprintf("Success! Processed: %s at %s", input, time.Now().Format(time.RFC3339))
	log.Printf("FlakyActivity succeeded: %s", result)
	
	return result, nil
}

// ActivityWithNonRetryableError demonstrates error types that should not be retried
func ActivityWithNonRetryableError(ctx context.Context, input string) (any, error) {
	log.Printf("ActivityWithNonRetryableError executing with input: %s", input)

	// Simulate different types of errors
	errorTypes := []string{
		"fatal error",        // Non-retryable
		"validation error",   // Non-retryable
		"temporary failure",  // Retryable
		"network timeout",    // Retryable
	}

	// Pick a random error type
	errorType := errorTypes[rand.Intn(len(errorTypes))]
	
	if errorType == "fatal error" || errorType == "validation error" {
		log.Printf("ActivityWithNonRetryableError failing with non-retryable error: %s", errorType)
		return nil, fmt.Errorf("%s", errorType)
	}

	// 50% chance of success for retryable scenarios
	if rand.Float64() < 0.5 {
		log.Printf("ActivityWithNonRetryableError failing with retryable error: %s", errorType)
		return nil, fmt.Errorf("%s", errorType)
	}

	result := fmt.Sprintf("Success! Input: %s", input)
	log.Printf("ActivityWithNonRetryableError succeeded: %s", result)
	
	return result, nil
}

// TimeoutExampleWorkflow demonstrates timeout configurations
func TimeoutExampleWorkflow(ctx workflow.Context, input string) (any, error) {
	log.Println("timeout example workflow started")

	// Activity with short timeout
	shortTimeoutOptions := &types.ActivityOptions{
		TaskQueue:              "default",
		ScheduleToCloseTimeout: 30 * time.Second,
		StartToCloseTimeout:    5 * time.Second,  // Very short timeout
		RetryPolicy: &types.RetryPolicy{
			MaximumAttempts: 2,
			InitialInterval: 1 * time.Second,
			BackoffPolicy:   types.BackoffFixed,
		},
	}

	var timeoutResult string
	err := workflow.ExecuteActivityWithOptions(ctx, SlowActivity, shortTimeoutOptions, input, 10*time.Second).
		Get(ctx, &timeoutResult)
	if err != nil {
		log.Printf("Activity timed out as expected: %v", err)
	}

	// Activity with reasonable timeout
	normalTimeoutOptions := &types.ActivityOptions{
		TaskQueue:              "default",
		ScheduleToCloseTimeout: 2 * time.Minute,
		StartToCloseTimeout:    30 * time.Second,
		RetryPolicy:            types.DefaultRetryPolicy(),
	}

	var normalResult string
	err = workflow.ExecuteActivityWithOptions(ctx, SlowActivity, normalTimeoutOptions, input, 2*time.Second).
		Get(ctx, &normalResult)
	if err != nil {
		return nil, fmt.Errorf("normal activity failed: %w", err)
	}

	result := map[string]any{
		"timeout_result": timeoutResult,
		"normal_result":  normalResult,
		"status":         "completed",
	}

	return result, nil
}

// SlowActivity simulates a slow-running activity
func SlowActivity(ctx context.Context, input string, duration time.Duration) (any, error) {
	log.Printf("SlowActivity executing for %v", duration)

	// Simulate work that takes the specified duration
	time.Sleep(duration)

	result := fmt.Sprintf("Slow processing completed for: %s", input)
	log.Printf("SlowActivity completed: %s", result)
	
	return result, nil
}