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
