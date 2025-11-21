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
	"github.com/ngnhng/durablefuture/sdk/internal"
)

const activityOptionsKey = internal.ActivityOptionsKey

// ActivityOptions configures how an activity is executed, including timeouts and retry behavior.
//
// Use WithActivityOptions to set these options:
//
//	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
//		ScheduleToCloseTimeout: 5 * time.Minute,
//		StartToCloseTimeout:    30 * time.Second,
//		RetryPolicy: &workflow.RetryPolicy{
//			InitialInterval:    time.Second,
//			BackoffCoefficient: 2.0,
//			MaximumAttempts:    3,
//		},
//	})
type ActivityOptions = internal.ActivityOptions

// RetryPolicy defines how activities are retried on failure.
//
// Retries use exponential backoff with configurable parameters. Activities are
// retried automatically unless:
//   - MaximumAttempts is reached
//   - The error type is in NonRetryableErrorTypes
//   - The activity context is canceled
//
// Example:
//
//	RetryPolicy: &workflow.RetryPolicy{
//		InitialInterval:    time.Second,      // First retry after 1s
//		BackoffCoefficient: 2.0,              // Double delay each retry
//		MaximumInterval:    30 * time.Second, // Cap delay at 30s
//		MaximumAttempts:    5,                // Give up after 5 attempts
//		NonRetryableErrorTypes: []string{
//			"invalid input",  // Don't retry validation errors
//		},
//	}
type RetryPolicy = internal.RetryPolicy

func WithActivityOptions(ctx Context, opts ActivityOptions) Context {
	return ctx.WithValue(activityOptionsKey, opts)
}
