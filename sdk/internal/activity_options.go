package internal

import (
	"math"
	"slices"
	"time"

	"github.com/ngnhng/durablefuture/api"
)

const ActivityOptionsKey = "github.com/ngnhng/durablefuture/sdk/workflow.ActivityOptions"

type ActivityOptions struct {
	// ScheduleToCloseTimeout is the total time allowed for the Activity during the entire Workflow Execution, including retries.
	//
	// The zero value of this uses default value of Unlimited.
	//
	// Either this option or StartToCloseTimeout is required. Defaults to Unlimited.
	ScheduleToCloseTimeout time.Duration

	// StartToCloseTimeout - Maximum time of a single Activity execution attempt.
	//
	// Note that the Server doesn't detect Worker process failures directly. It relies on this timeout
	// to detect that an Activity that didn't complete on time. So this timeout should be as short as the longest
	// possible execution of the Activity body. Potentially long-running Activities must specify HeartbeatTimeout
	// and call Activity.RecordHeartbeat(ctx, "my-heartbeat") periodically for timely failure detection.
	//
	// Either this option or ScheduleToCloseTimeout is required. Defaults to the ScheduleToCloseTimeout value.
	StartToCloseTimeout time.Duration

	RetryPolicy *RetryPolicy
}

type RetryPolicy struct {
	// Backoff interval for the first retry. If BackoffCoefficient is 1.0 then it is used for all retries.
	// If not set or set to 0, a default interval of 1s will be used.
	InitialInterval time.Duration

	// Coefficient used to calculate the next retry backoff interval.
	// The next retry interval is previous interval multiplied by this coefficient.
	// Must be 1 or larger. Default is 2.0.
	BackoffCoefficient float64

	// Maximum backoff interval between retries. Exponential backoff leads to interval increase.
	// This value is the cap of the interval. Default is 100x of initial interval.
	MaximumInterval time.Duration

	// Maximum number of attempts. When exceeded the retries stop even if not expired yet.
	// If not set or set to 0, it means unlimited, and rely on activity ScheduleToCloseTimeout to stop.
	MaximumAttempts int32

	// Non-Retryable errors. This is optional. Server will stop retry if error type matches this list.
	//
	// Note:
	//  - cancellation is not a failure, so it won't be retried,
	//  - only StartToClose or Heartbeat timeouts are retryable.
	NonRetryableErrorTypes []string
}

// CalculateNextDelay calculates the next retry delay using exponential backoff
func (r *RetryPolicy) CalculateNextDelay(attempt int64) time.Duration {
	nextDelay := time.Duration(
		float64(r.InitialInterval) *
			math.Pow(
				r.BackoffCoefficient,
				float64(attempt)-1),
	)

	return nextDelay
}

// ShouldRetry determines if an error should be retried based on the retry policy
func (r *RetryPolicy) ShouldRetry(attempt int, err error) bool {
	if r.MaximumAttempts >= 0 && attempt <= int(r.MaximumAttempts) {
		if slices.Contains(r.NonRetryableErrorTypes, err.Error()) {
			return true
		}
	}

	return false
}

func getActivityOptions(ctx Context) *ActivityOptions {
	val := ctx.Value(ActivityOptionsKey)
	if val == nil {
		return nil
	}

	opts, ok := val.(ActivityOptions)
	if !ok {
		panic("ActivityOptions has wrong type in context.")
	}
	return &opts
}

func convertRetryPolicyToAPI(rp *RetryPolicy) *api.RetryPolicy {
	if rp == nil {
		return nil
	}
	return &api.RetryPolicy{
		InitialIntervalMs:      rp.InitialInterval.Milliseconds(),
		BackoffCoefficient:     rp.BackoffCoefficient,
		MaximumIntervalMs:      rp.MaximumInterval.Milliseconds(),
		MaximumAttempts:        rp.MaximumAttempts,
		NonRetryableErrorTypes: rp.NonRetryableErrorTypes,
	}
}
