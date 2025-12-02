package internal

import (
	"context"
	"fmt"
	"iter"
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
	"github.com/ngnhng/durablefuture/sdk/internal/common"
	"github.com/ngnhng/durablefuture/sdk/internal/protocol"
	"golang.org/x/sync/errgroup"
)

// --- task processor: worker runtime processing nats incoming messages ---
type taskProcessor interface {
	ReceiveTask(ctx context.Context, includeWorkflow, includeActivity bool) (iter.Seq[*protocol.TaskToken], error)
}

var _ taskProcessor = (*worker)(nil)

// --- kv store ----
type kv interface {
	get(k string) (any, error)
	set(k string, v any) error
	size() int64
}

// --- workflow registry ---
type (
	WorkflowRegisterOption struct{}

	WorkflowRegistry interface {
		RegisterWorkflow(w any, options ...WorkflowRegisterOption) error
	}
)

var _ WorkflowRegistry = (*worker)(nil)

// --- activity registry ---
type (
	ActivityRegisterOption struct{}

	ActivityRegistry interface {
		RegisterActivity(a any, options ...ActivityRegisterOption) error
	}
)

var _ ActivityRegistry = (*worker)(nil)

// --- concrete implementation of the registries and task processor ---
type WorkerOptions struct {
	TenantID  string
	Namespace string
	Logger    *slog.Logger
}
type worker struct {
	c client

	serder        serde.BinarySerde
	typeConverter *serde.TypeConverter

	// the sdk nats connection implements this
	taskProcessor

	workflowRegistry kv
	activityRegistry kv

	eventLog *eventlog.NATS
	logger   *slog.Logger
}

func NewWorker(c Client, opts *WorkerOptions) (*worker, error) {
	// Type assert to get internal methods
	ci, ok := c.(client)
	if !ok {
		return nil, fmt.Errorf("client does not implement required internal interface")
	}
	if opts == nil {
		opts = &WorkerOptions{}
	}

	logger := opts.Logger
	if logger == nil && ci != nil {
		logger = ci.getLogger()
	}
	logger = common.DefaultLogger(logger)

	streamName := buildHistoryStreamName(opts)
	subjectPrefix := opts.Namespace
	if subjectPrefix == "" {
		subjectPrefix = api.HistorySubjectPrefix
	}

	// TODO: api based config
	log, err := eventlog.NewNATSJetStream(
		ci.getConn().NATS(),
		eventlog.WithNATSStreamName(streamName),
		eventlog.WithNATSSubjectPrefix(subjectPrefix),
	)
	if err != nil {
		return nil, err
	}

	serder := ci.getSerde()
	return &worker{
		c:                ci,
		serder:           serder,
		typeConverter:    serde.NewTypeConverter(serder),
		taskProcessor:    ci.getConn(),
		eventLog:         log,
		workflowRegistry: newInMemoryRegistry(),
		activityRegistry: newInMemoryRegistry(),
		logger:           logger,
	}, nil
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

func (w *worker) RegisterWorkflow(fn any, options ...WorkflowRegisterOption) error {
	fnName, err := common.ExtractFullFunctionName(fn)
	if err != nil {
		return err
	}

	err = w.workflowRegistry.set(fnName, fn)
	if err != nil {
		return err
	}

	return nil
}

func (w *worker) RegisterActivity(fn any, opts ...ActivityRegisterOption) error {
	fnName, err := common.ExtractFullFunctionName(fn)
	if err != nil {
		return err
	}
	err = w.activityRegistry.set(fnName, fn)
	if err != nil {
		return err
	}

	return nil
}

func (w *worker) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		// The processing loop manages its own group of goroutines for each task.
		return w.runProcessingLoop(gCtx)
	})

	return g.Wait()
}

func (w *worker) runProcessingLoop(ctx context.Context) error {
	repo, err := chronicle.NewEventSourcedRepository(
		w.eventLog,
		newEmptyWorkflowContext,
		nil,
		aggregate.EventSerializer(w.serder),
	)
	if err != nil {
		w.logger.Error("cannot create repository, sending NAK", "error", err)
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
					workflow, err := repo.Get(gCtx, api.WorkflowID(task.WorkflowID))
					if err != nil {
						w.logger.Error("failed to replay workflow", "workflow_id", task.WorkflowID, "error", err)
						// Terminate the task if we can't even replay the workflow.
						token.Term(gCtx)
						return err
					}

					workflow.Context = gCtx
					workflow.serder = w.serder
					workflow.logger = w.logger

					err = w.processWorkflowTask(workflow, task)
					if err != nil {
						w.logger.Error("workflow task failed, sending NAK", "workflow_id", task.WorkflowID, "error", err)
						token.Nak(gCtx)
					} else {
						w.logger.Debug("workflow task succeeded, sending ACK", "workflow_id", task.WorkflowID)
						token.Ack(gCtx)
					}

					_, _, err = repo.Save(gCtx, workflow)
					if err != nil {
						return err
					}
				}

			case *api.ActivityTask:
				{
					wfctx, err := repo.Get(gCtx, api.WorkflowID(task.WorkflowID))
					if err != nil {
						w.logger.Error("failed to replay activity", "workflow_id", task.WorkflowID, "error", err)
						// Terminate the task if we can't even replay the workflow.
						token.Term(gCtx)
						return err
					}

					wfctx.Context = gCtx
					wfctx.serder = w.serder
					wfctx.logger = w.logger

					err = w.processActivityTask(wfctx, task)
					if err != nil {
						w.logger.Error("activity task failed, sending NAK", "workflow_id", task.WorkflowID, "activity", task.ActivityFn, "error", err)
						token.Nak(gCtx)
					} else {
						w.logger.Debug("activity task succeeded, sending ACK", "workflow_id", task.WorkflowID, "activity", task.ActivityFn)
						token.Ack(gCtx)
					}

					_, _, err = repo.Save(gCtx, wfctx)
					if err != nil {
						return err
					}
				}
			default:
				// poison pill
				w.logger.Warn("received poison pill, terminating task")
				token.Term(gCtx)
			}

			return nil
		})
	}

	return g.Wait()
}

func (w *worker) processActivityTask(wfctx *workflowContext, task *api.ActivityTask) error {
	fn, err := w.activityRegistry.get(task.ActivityFn)
	if err != nil {
		w.logger.Error("activity not found in registry", "activity", task.ActivityFn, "error", err)
		return err
	}

	result, err := w.executeActivityFunc(wfctx, fn, task.Input)
	if err != nil {
		w.logger.Warn("activity execution failed", "activity", task.ActivityFn, "attempt", task.Attempt, "error", err)

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

			w.logger.Info("activity will retry", "activity", task.ActivityFn, "attempt", task.Attempt, "next_delay", nextDelay)
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
		Result:         common.ReflectValuesToAny(result),
	})

	return nil
}

// evaluateRetryDecision determines if an activity should be retried based on retry policy
func (w *worker) evaluateRetryDecision(task *api.ActivityTask, err error) bool {
	// No retry policy means no retries
	if task.RetryPolicy == nil {
		return false
	}

	policy := task.RetryPolicy

	// Check if max attempts exceeded
	if policy.MaximumAttempts > 0 && task.Attempt >= policy.MaximumAttempts {
		w.logger.Info("max attempts reached for activity", "activity", task.ActivityFn, "max_attempts", policy.MaximumAttempts)
		return false
	}

	// Check if error is non-retryable
	if len(policy.NonRetryableErrorTypes) > 0 && slices.Contains(policy.NonRetryableErrorTypes, err.Error()) {
		w.logger.Info("non-retryable error for activity", "activity", task.ActivityFn, "error", err.Error())
		return false
	}

	// Check if schedule-to-close timeout would be exceeded
	if task.ScheduleToCloseTimeoutUnix > 0 {
		elapsedMs := time.Now().UnixMilli() - task.ScheduledAtMs
		nextDelay := w.calculateRetryDelay(task)
		if elapsedMs+nextDelay.Milliseconds() > task.ScheduleToCloseTimeoutUnix {
			w.logger.Info("schedule-to-close timeout would be exceeded for activity", "activity", task.ActivityFn)
			return false
		}
	}

	return true
}

// calculateRetryDelay calculates the backoff delay for the next retry attempt
func (w *worker) calculateRetryDelay(task *api.ActivityTask) time.Duration {
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

func (w *worker) processWorkflowTask(wfctx *workflowContext, task *api.WorkflowTask) error {
	var results []reflect.Value
	var pending bool
	var panicked bool

	execErr := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				if _, ok := r.(errorBlockingFuture); ok {
					pending = true
					return
				}
				w.logger.Error("workflow execution panic", "workflow_id", task.WorkflowID, "panic", r)
				err = fmt.Errorf("workflow panic: %v", r)
			}
		}()

		fn, err := w.workflowRegistry.get(task.WorkflowFn)
		if err != nil {
			w.logger.Error("workflow function lookup failed", "workflow", task.WorkflowFn, "error", err)
			return err
		}
		wfv := reflect.ValueOf(fn)
		wft := wfv.Type()
		inputv := make([]reflect.Value, len(task.Input))
		w.logger.Debug("workflow input received", "workflow_id", task.WorkflowID, "input_len", len(task.Input))
		for idx, arg := range task.Input {
			// Skip the first parameter which is the context
			paramType := wft.In(idx + 1)
			convertedArg, err := w.convertToType(arg, paramType)
			if err != nil {
				w.logger.Debug("failed to convert workflow parameter", "workflow_id", task.WorkflowID, "param_index", idx, "error", err)
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
			Result:         common.ReflectValuesToAny(results),
		})
		return nil
	}
}

func (w *worker) executeActivityFunc(ctx context.Context, fn any, inputs []any) (result []reflect.Value, err error) {
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
func (w *worker) convertToType(value any, targetType reflect.Type) (reflect.Value, error) {
	return w.typeConverter.ConvertToType(value, targetType)
}
