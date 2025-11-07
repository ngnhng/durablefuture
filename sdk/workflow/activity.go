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
	"math"
	"slices"
	"time"
)

type activityOptionsKey struct{}

type (
	ActivityOptions struct {
		// ScheduleToCloseTimeout is the total time allowed for the Activity during the entire Workflow Execution, including retries
		// The zero value of this uses default value of Unlimited.
		// Either this option or StartToCloseTimeout is required: Defaults to Unlimited.
		ScheduleToCloseTimeout time.Duration

		// StartToCloseTimeout - Maximum time of a single Activity execution attempt.
		// Note that the Temporal Server doesn't detect Worker process failures directly. It relies on this timeout
		// to detect that an Activity that didn't complete on time. So this timeout should be as short as the longest
		// possible execution of the Activity body. Potentially long-running Activities must specify HeartbeatTimeout
		// and call Activity.RecordHeartbeat(ctx, "my-heartbeat") periodically for timely failure detection.
		// Either this option or ScheduleToCloseTimeout is required: Defaults to the ScheduleToCloseTimeout value.
		StartToCloseTimeout time.Duration

		RetryPolicy *RetryPolicy
	}

	RetryPolicy struct {
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

		// Non-Retriable errors. This is optional. Temporal server will stop retry if error type matches this list.
		//
		// Note:
		//  - cancellation is not a failure, so it won't be retried,
		//  - only StartToClose or Heartbeat timeouts are retryable.
		NonRetryableErrorTypes []string
	}
)

func (r *RetryPolicy) CalculateNextDelay(attempt int64) time.Duration {
	nextDelay := time.Duration(
		float64(r.InitialInterval) *
			(math.Pow(
				r.BackoffCoefficient,
				float64(attempt)-1)),
	)

	return nextDelay
}

func (r *RetryPolicy) ShouldRetry(attempt int, err error) bool {
	if r.MaximumAttempts >= 0 && attempt <= int(r.MaximumAttempts) {
		if slices.Contains(r.NonRetryableErrorTypes, err.Error()) {
			return true
		}
	}

	return false
}

func WithActivityOptions(ctx Context, opts ActivityOptions) Context {
	return ctx.WithValue(activityOptionsKey{}, opts)
}
