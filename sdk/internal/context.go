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

package internal

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/DeluxeOwl/chronicle/aggregate"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/sdk/internal/utils"
)

type Context interface {
	context.Context
	ExecuteActivity(activityFn any, args ...any) Future
	ID() api.WorkflowID
	GetWorkflowFunctionName() string
	WithValue(key any, value any) Context
}

var _ Context = (*workflowContext)(nil)

type workflowContext struct {
	*workflowState
	context.Context
	logger *slog.Logger
}

type workflowState struct {
	aggregate.Base
	converter            serde.BinarySerde
	id                   api.WorkflowID
	workflowFunctionName string
	activities           map[string]*activityReplay
}

type activityReplay struct {
	history  []activityReplayRecord
	consumed int
}

type activityReplayRecord struct {
	result []any
	err    error
}

func NewEmptyWorkflowContext() *workflowContext {
	return NewWorkflowContextWithLogger(nil)
}

func NewWorkflowContextWithLogger(logger *slog.Logger) *workflowContext {
	return &workflowContext{
		workflowState: &workflowState{
			activities: make(map[string]*activityReplay),
		},
		Context: context.Background(),
		logger:  defaultLogger(logger),
	}
}

func (c *workflowContext) ensureActivityReplay(fnName string) *activityReplay {
	if c.activities == nil {
		c.activities = make(map[string]*activityReplay)
	}
	if entry, ok := c.activities[fnName]; ok {
		return entry
	}
	entry := &activityReplay{}
	c.activities[fnName] = entry
	return entry
}

func (c *workflowContext) ExecuteActivity(activityFn any, args ...any) Future {
	fnName, err := utils.ExtractFullFunctionName(activityFn)
	if err != nil { /* panic */
		c.loggerOrDefault().Error("failed to extract activity function name", "error", err)
		panic(err)
	}
	entry := c.ensureActivityReplay(fnName)

	// During replay we look for an already completed/failed result in order.
	if entry.consumed < len(entry.history) {
		record := entry.history[entry.consumed]
		entry.consumed++
		if record.err != nil {
			return &pending{isResolved: true, err: record.err, converter: c.converter, logger: c.logger}
		}
		return &pending{isResolved: true, value: record.result, converter: c.converter, logger: c.logger}
	}

	// No cached history entry means this is a fresh execution.
	entry.consumed++

	// Read activity options from context if available
	opts := getActivityOptions(c)
	event := &api.ActivityScheduled{
		ID:             c.ID(),
		WorkflowFnName: c.GetWorkflowFunctionName(),
		ActivityFnName: fnName,
		Input:          args,
	}

	// Populate timeout and retry policy from options
	if opts != nil {
		if opts.ScheduleToCloseTimeout > 0 {
			event.ScheduleToCloseTimeoutMs = opts.ScheduleToCloseTimeout.Milliseconds()
		}
		if opts.StartToCloseTimeout > 0 {
			event.StartToCloseTimeoutMs = opts.StartToCloseTimeout.Milliseconds()
		}
		if opts.RetryPolicy != nil {
			event.RetryPolicy = convertRetryPolicyToAPI(opts.RetryPolicy)
		}
	}

	if err := c.recordThat(event); err != nil {
		// recording should never fail during workflow execution; surface loudly if it does
		panic(fmt.Errorf("record activity scheduled event: %w", err))
	}

	return &pending{isResolved: false, converter: c.converter, logger: c.logger}
}

const ActivityOptionsKey = "github.com/ngnhng/durablefuture/sdk/workflow.ActivityOptions"

type ActivityOptions struct {
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

	// Non-Retryable errors. This is optional. Temporal server will stop retry if error type matches this list.
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

func (c *workflowContext) WithValue(key any, value any) Context {
	baseCtx := c.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	return &workflowContext{
		workflowState: c.workflowState,
		Context:       context.WithValue(baseCtx, key, value),
		logger:        c.logger,
	}
}

func (c *workflowContext) loggerOrDefault() *slog.Logger {
	if c == nil {
		return slog.Default()
	}
	return defaultLogger(c.logger)
}

// getNewEvents is an unexported method accessible only within the `workflow` package.
// The executor calls this to retrieve the commands generated by the workflow logic.
func (c *workflowContext) GetWorkflowFunctionName() string { return c.workflowFunctionName }

func (c *workflowContext) Deadline() (time.Time, bool) {
	if c.Context == nil {
		return time.Time{}, false
	}
	return c.Context.Deadline()
}

func (c *workflowContext) Done() <-chan struct{} {
	if c.Context == nil {
		return nil
	}
	return c.Context.Done()
}

func (c *workflowContext) Err() error {
	if c.Context == nil {
		return nil
	}
	return c.Context.Err()
}

func (c *workflowContext) Value(key any) any {
	if c.Context == nil {
		return nil
	}
	return c.Context.Value(key)
}
