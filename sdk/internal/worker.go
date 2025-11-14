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
	"iter"
	"log"
	"log/slog"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/DeluxeOwl/chronicle"
	"github.com/DeluxeOwl/chronicle/aggregate"
	"github.com/DeluxeOwl/chronicle/eventlog"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/sdk/internal/utils"
	"golang.org/x/sync/errgroup"
)

type (
	TaskToken struct {
		Task api.Task
		Ack  func(context.Context) error
		Nak  func(context.Context) error
		Term func(context.Context) error
	}

	TaskProcessor interface {
		ReceiveTask(ctx context.Context, includeWorkflow, includeActivity bool) (iter.Seq[*TaskToken], error)
	}

	WorkerOptions struct {
		TenantID  string
		Namespace string
	}

	ActivityRegisterOption struct{}

	WorkflowRegisterOption struct{}

	WorkflowRegistry interface {
		RegisterWorkflow(w any, options ...WorkflowRegisterOption) error
	}

	ActivityRegistry interface {
		RegisterActivity(a any, options ...ActivityRegisterOption) error
	}
)
type workerImpl struct {
	c Client

	converter     serde.BinarySerde
	typeConverter *serde.TypeConverter // for serialization-agnostic type conversion

	TaskProcessor

	workflowRegistry registry
	activityRegistry registry

	evtLog *eventlog.NATS
}

func NewWorker(c Client, opts *WorkerOptions) (*workerImpl, error) {
	if opts == nil {
		opts = &WorkerOptions{}
	}

	streamName := buildHistoryStreamName(opts)
	subjectPrefix := opts.Namespace
	if subjectPrefix == "" {
		subjectPrefix = api.HistorySubjectPrefix
	}

	// TODO: api based config
	log, err := eventlog.NewNATSJetStream(
		c.getConn().NATS(),
		eventlog.WithNATSStreamName(streamName),
		eventlog.WithNATSSubjectPrefix(subjectPrefix),
	)
	if err != nil {
		return nil, err
	}

	conv := c.getSerde()
	return &workerImpl{
		c:                c,
		converter:        conv,
		typeConverter:    serde.NewTypeConverter(conv),
		TaskProcessor:    c.getConn(),
		evtLog:           log,
		workflowRegistry: NewInMemoryRegistry(),
		activityRegistry: NewInMemoryRegistry(),
	}, nil
}

func (w *workerImpl) RegisterWorkflow(fn any, options ...WorkflowRegisterOption) error {
	fnName, err := utils.ExtractFullFunctionName(fn)
	if err != nil {
		return err
	}

	err = w.workflowRegistry.set(fnName, fn)
	if err != nil {
		return err
	}

	return nil
}

func (w *workerImpl) RegisterActivity(fn any, opts ...ActivityRegisterOption) error {
	fnName, err := utils.ExtractFullFunctionName(fn)
	if err != nil {
		return err
	}
	err = w.activityRegistry.set(fnName, fn)
	if err != nil {
		return err
	}

	return nil
}

func buildHistoryStreamName(opts *WorkerOptions) string {
	if opts == nil {
		return api.WorkflowHistoryStream
	}

	var parts []string
	if opts.TenantID != "" {
		parts = append(parts, opts.TenantID)
	}
	if opts.Namespace != "" {
		parts = append(parts, opts.Namespace)
	}
	parts = append(parts, api.WorkflowHistoryStream)
	return strings.Join(parts, "_")
}

func (w *workerImpl) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		// The processing loop now manages its own group of goroutines for each task.
		return w.runProcessingLoop(gCtx)
	})

	return g.Wait()
}

func (w *workerImpl) runProcessingLoop(ctx context.Context) error {
	// Pass the serializer to Chronicle so it uses MessagePack (or whatever serde is configured)
	repo, err := chronicle.NewEventSourcedRepository(
		w.evtLog,
		NewEmptyWorkflowContext,
		nil,
		aggregate.EventSerializer(w.converter),
	)
	if err != nil {
		log.Printf("Cannot create repository, sending Nak: %v", err)
		return err
	}

	// Create a new errgroup to manage a goroutine for each task.
	g, gCtx := errgroup.WithContext(ctx)

	workflowTasksEnabled := w.workflowRegistry.size() > 0
	activityTasksEnabled := w.activityRegistry.size() > 0
	if !workflowTasksEnabled && !activityTasksEnabled {
		return fmt.Errorf("worker has no registered workflows or activities")
	}

	iter, err := w.ReceiveTask(gCtx, workflowTasksEnabled, activityTasksEnabled)
	if err != nil {
		return err
	}
	for token := range iter {
		// Spawn a new goroutine for each received task.
		g.Go(func() error {
			switch task := token.Task.(type) {
			case *api.WorkflowTask:
				{
					wfctx, err := repo.Get(gCtx, api.WorkflowID(task.WorkflowID))
					if err != nil {
						log.Printf("Failed to replay workflow: %v", err)
						// Terminate the task if we can't even replay the workflow.
						token.Term(gCtx)
						return err
					}

					wfctx.Context = gCtx
					wfctx.converter = w.converter

					err = w.processWorkflowTask(wfctx, task)
					if err != nil {
						log.Printf("Workflow task failed, sending Nak: %v", err)
						token.Nak(gCtx)
					} else {
						log.Printf("Workflow task succeeded, sending Ack")
						token.Ack(gCtx)
					}

					_, _, err = repo.Save(gCtx, wfctx)
					if err != nil {
						return err
					}
				}

			case *api.ActivityTask:
				{
					wfctx, err := repo.Get(gCtx, api.WorkflowID(task.WorkflowID))
					if err != nil {
						log.Printf("Failed to replay activity: %v", err)
						// Terminate the task if we can't even replay the workflow.
						token.Term(gCtx)
						return err
					}

					wfctx.Context = gCtx
					wfctx.converter = w.converter

					err = w.processActivityTask(wfctx, task)
					if err != nil {
						log.Printf("Activity task failed, sending Nak: %v", err)
						token.Nak(gCtx)
					} else {
						log.Printf("Activity task succeeded, sending Ack")
						token.Ack(gCtx)
					}

					_, _, err = repo.Save(gCtx, wfctx)
					if err != nil {
						return err
					}
				}
			default:
				// poison pill
				log.Println("got poison pill")
				token.Term(gCtx)
			}

			return nil
		})
	}

	return g.Wait()
}

func (w *workerImpl) processActivityTask(wfctx *workflowContext, task *api.ActivityTask) error {
	fn, err := w.activityRegistry.get(task.ActivityFn)
	if err != nil {
		log.Printf("failed to get %s from activity registry", task.ActivityFn)
		return err
	}

	result, err := w.executeActivityFunc(wfctx, fn, task.Input)
	if err != nil {
		log.Printf("Activity %s failed (attempt %d): %v", task.ActivityFn, task.Attempt, err)

		// Check if we should retry based on the retry policy
		if shouldRetry := w.evaluateRetryDecision(task, err); shouldRetry {
			// Calculate next retry delay
			nextDelay := w.calculateRetryDelay(task)

			// Record retry event
			wfctx.recordThat(&api.ActivityRetried{
				ID:             api.WorkflowID(task.WorkflowID),
				WorkflowFnName: task.WorkflowFn,
				ActivityFnName: task.ActivityFn,
				Attempt:        task.Attempt,
				Error:          err.Error(),
				NextRetryDelay: nextDelay.Milliseconds(),
			})

			log.Printf("Activity %s failed (attempt %d), will retry after %v", task.ActivityFn, task.Attempt, nextDelay)
			return nil
		}

		// Not retrying, record final failure
		wfctx.recordThat(&api.ActivityFailed{
			ID:             api.WorkflowID(task.WorkflowID),
			ActivityFnName: task.ActivityFn,
			WorkflowFnName: task.WorkflowFn,
			Error:          err.Error(),
		})

		return nil
	}

	// Success - record completion
	wfctx.recordThat(&api.ActivityCompleted{
		ID:             api.WorkflowID(task.WorkflowID),
		WorkflowFnName: task.WorkflowFn,
		ActivityFnName: task.ActivityFn,
		Result:         reflectValuesToAny(result),
	})

	return nil
}

// evaluateRetryDecision determines if an activity should be retried based on retry policy
func (w *workerImpl) evaluateRetryDecision(task *api.ActivityTask, err error) bool {
	// No retry policy means no retries
	if task.RetryPolicy == nil {
		return false
	}

	policy := task.RetryPolicy

	// Check if max attempts exceeded
	if policy.MaximumAttempts > 0 && task.Attempt >= policy.MaximumAttempts {
		log.Printf("Max attempts (%d) reached for activity %s", policy.MaximumAttempts, task.ActivityFn)
		return false
	}

	// Check if error is non-retryable
	if len(policy.NonRetryableErrorTypes) > 0 && slices.Contains(policy.NonRetryableErrorTypes, err.Error()) {
		log.Printf("Error '%s' is non-retryable for activity %s", err.Error(), task.ActivityFn)
		return false
	}

	// Check if schedule-to-close timeout would be exceeded
	if task.ScheduleToCloseTimeoutMs > 0 {
		elapsedMs := time.Now().UnixMilli() - task.ScheduledAtMs
		nextDelay := w.calculateRetryDelay(task)
		if elapsedMs+nextDelay.Milliseconds() > task.ScheduleToCloseTimeoutMs {
			log.Printf("ScheduleToCloseTimeout would be exceeded for activity %s", task.ActivityFn)
			return false
		}
	}

	return true
}

// calculateRetryDelay calculates the backoff delay for the next retry attempt
func (w *workerImpl) calculateRetryDelay(task *api.ActivityTask) time.Duration {
	if task.RetryPolicy == nil {
		return time.Second // Default 1 second
	}

	policy := task.RetryPolicy

	// Set defaults
	initialInterval := time.Duration(policy.InitialIntervalMs) * time.Millisecond
	if initialInterval == 0 {
		initialInterval = time.Second
	}

	backoffCoefficient := policy.BackoffCoefficient
	if backoffCoefficient <= 0 {
		backoffCoefficient = 2.0
	}

	maxInterval := time.Duration(policy.MaximumIntervalMs) * time.Millisecond
	if maxInterval == 0 {
		maxInterval = 100 * initialInterval
	}

	// Calculate exponential backoff: initialInterval * (backoffCoefficient ^ (attempt - 1))
	nextDelay := time.Duration(
		float64(initialInterval) * math.Pow(backoffCoefficient, float64(task.Attempt-1)),
	)

	// Cap at maximum interval
	nextDelay = min(nextDelay, maxInterval)

	return nextDelay
}

func (w *workerImpl) processWorkflowTask(wfctx *workflowContext, task *api.WorkflowTask) error {
	var results []reflect.Value
	var pending bool
	var panicked bool

	execErr := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				if _, ok := r.(ErrorBlockingFuture); ok {
					pending = true
					return
				}
				log.Printf("panic but not a blocking future: %v\n", r)
				err = fmt.Errorf("workflow panic: %v", r)
			}
		}()

		fn, err := w.workflowRegistry.get(task.WorkflowFn)
		if err != nil {
			log.Printf("cannot get workflow function: %v\n", err)
			return err
		}
		wfv := reflect.ValueOf(fn)
		wft := wfv.Type()
		inputv := make([]reflect.Value, len(task.Input))
		slog.Debug(fmt.Sprintf("input params: %v with length: %v", task.Input, len(task.Input)))
		for idx, arg := range task.Input {
			// Skip the first parameter which is the context
			paramType := wft.In(idx + 1)
			convertedArg, err := w.convertToType(arg, paramType)
			if err != nil {
				slog.Debug(fmt.Sprintf("Failed to convert parameter %d: %v", idx, err))
				return err
			}
			inputv[idx] = convertedArg
		}

		results = wfv.Call(append([]reflect.Value{reflect.ValueOf(wfctx)}, inputv...))

		return nil
	}()

	switch {
	case panicked && pending:
		return nil
	case panicked && !pending:
		if execErr == nil {
			execErr = fmt.Errorf("workflow execution panicked")
		}
		wfctx.recordThat(&api.WorkflowFailed{
			ID:             api.WorkflowID(task.WorkflowID),
			WorkflowFnName: task.WorkflowFn,
			Error:          execErr.Error(),
		})
		return execErr
	case !panicked && execErr != nil:
		wfctx.recordThat(&api.WorkflowFailed{
			ID:             api.WorkflowID(task.WorkflowID),
			WorkflowFnName: task.WorkflowFn,
			Error:          execErr.Error(),
		})
		return execErr
	default:
		wfctx.recordThat(&api.WorkflowCompleted{
			ID:             api.WorkflowID(task.WorkflowID),
			WorkflowFnName: task.WorkflowFn,
			Result:         reflectValuesToAny(results),
		})
		return nil
	}
}

func reflectValuesToAny(vals []reflect.Value) []any {
	anySlice := make([]any, len(vals))
	for i, v := range vals {
		anySlice[i] = v.Interface()
	}
	return anySlice
}

func (w *workerImpl) executeActivityFunc(ctx context.Context, fn any, inputs []any) (result []reflect.Value, err error) {
	fnv := reflect.ValueOf(fn)
	fnt := fnv.Type()

	if fnt.NumIn() != len(inputs)+1 { // +1 for the context.Context
		return nil, fmt.Errorf("argument count mismatch: activity expects %d, got %d", fnt.NumIn()-1, len(inputs))
	}
	if fnt.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return nil, fmt.Errorf("activity function must accept context.Context as its first argument")
	}

	callArgs := make([]reflect.Value, len(inputs)+1)
	callArgs[0] = reflect.ValueOf(ctx)
	for idx, arg := range inputs {
		// Skip the first parameter which is the context
		paramType := fnt.In(idx + 1)
		convertedArg, err := w.convertToType(arg, paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter %d: %v", idx, err)
		}
		callArgs[idx+1] = convertedArg
	}

	rawResults := fnv.Call(callArgs)

	// Check if the last return value is an error
	if len(rawResults) > 0 {
		lastResult := rawResults[len(rawResults)-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				err = lastResult.Interface().(error)
			}
		}
	}

	return rawResults, err
}

// convertToType converts a value to the target type using serialization-agnostic approach.
// This delegates to the TypeConverter which uses the configured BinarySerde,
// making it work regardless of whether we're using JSON, msgpack, protobuf, etc.
func (w *workerImpl) convertToType(value any, targetType reflect.Type) (reflect.Value, error) {
	return w.typeConverter.ConvertToType(value, targetType)
}
