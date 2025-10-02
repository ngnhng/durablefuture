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
	"encoding/json"
	"fmt"
	"iter"
	"log"
	"log/slog"
	"reflect"

	"github.com/DeluxeOwl/chronicle"
	"github.com/DeluxeOwl/chronicle/eventlog"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/server/utils"
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
		ReceiveTask(ctx context.Context) (iter.Seq[*TaskToken], error)
	}

	WorkerOptions struct {
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

	converter serde.BinarySerde

	TaskProcessor

	workflowRegistry registry
	activityRegistry registry

	evtLog *eventlog.NATS
}

func NewWorker(c Client, opts *WorkerOptions) (*workerImpl, error) {

	// TODO: api based config
	log, err := eventlog.NewNATSJetStream(c.getConn().nc, eventlog.WithNATSStreamName("HISTORY"))
	if err != nil {
		return nil, err
	}

	return &workerImpl{
		c:                c,
		converter:        c.getSerde(),
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

func (w *workerImpl) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		// The processing loop now manages its own group of goroutines for each task.
		return w.runProcessingLoop(gCtx)
	})

	return g.Wait()
}

func (w *workerImpl) runProcessingLoop(ctx context.Context) error {
	repo, err := chronicle.NewEventSourcedRepository(
		w.evtLog,
		NewEmptyWorkflowContext,
		nil,
	)
	if err != nil {
		log.Printf("Cannot create repository, sending Nak: %v", err)
		return err
	}

	// Create a new errgroup to manage a goroutine for each task.
	g, gCtx := errgroup.WithContext(ctx)

	iter, err := w.ReceiveTask(gCtx)
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
		log.Printf("failed to execute %s with %s from activity registry", task.ActivityFn, task.Input)

		wfctx.recordThat(&api.ActivityFailed{
			ID:             api.WorkflowID(task.WorkflowID),
			ActivityFnName: task.ActivityFn,
			WorkflowFnName: task.WorkflowFn,
			Error:          err.Error(),
		})
	}

	wfctx.recordThat(&api.ActivityCompleted{
		ID:             api.WorkflowID(task.WorkflowID),
		WorkflowFnName: task.WorkflowFn,
		ActivityFnName: task.ActivityFn,
		Result:         reflectValuesToAny(result),
	})

	return nil

}

func (w *workerImpl) processWorkflowTask(wfctx *workflowContext, task *api.WorkflowTask) error {
	var results []reflect.Value
	var pending bool
	var panicked bool

	err := func() error {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(ErrorBlockingFuture); !ok {
					log.Printf("panic but not a blocking future: %v\n", r)
				}
			} else {
				pending = true
			}
			panicked = true
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

	if panicked {
		if !pending {
			wfctx.recordThat(&api.WorkflowFailed{
				ID:             api.WorkflowID(task.WorkflowID),
				WorkflowFnName: task.WorkflowFn,
				Error:          err.Error(),
			})
		}
	} else {
		wfctx.recordThat(&api.WorkflowCompleted{
			ID:             api.WorkflowID(task.WorkflowID),
			WorkflowFnName: task.WorkflowFn,
			// TODO: document this
			Result: reflectValuesToAny(results),
		})
	}

	return err
}

func reflectValuesToAny(vals []reflect.Value) []any {
	anySlice := make([]any, len(vals))
	for i, v := range vals {
		anySlice[i] = v.Interface()
	}
	return anySlice
}

// workerTaskConsumer, err := js.CreateOrUpdateConsumer(ctx, api.WorkflowTasksStream, jetstream.ConsumerConfig{
// 	Durable:       api.WorkflowTaskWorkerConsumer,
// 	AckPolicy:     jetstream.AckExplicitPolicy,
// 	FilterSubject: api.WorkflowTasksFilterSubjectPattern,
// })
// if err != nil {
// 	slog.Debug(fmt.Sprintf("[WORKER]: %v", err))
// 	return fmt.Errorf("error: %v", err)
// }

// wc, err := workerTaskConsumer.Consume(func(msg jetstream.Msg) {
// 	slog.Debug(fmt.Sprintf("[WORKER] received: %v", string(msg.Data())))

// 	workflowWorkerID := uuid.Must(uuid.NewV7())
// 	if err != nil {
// 		slog.Debug(fmt.Sprintf("[WORKER]: %v", err))

// 		log.Fatalf("Worker %v failed to create consumer: %v", workflowWorkerID, err)
// 	}

// 	var task api.WorkflowTask
// 	if err := json.Unmarshal(msg.Data(), &task); err != nil {
// 		slog.Debug(fmt.Sprintf("Worker %v: could not unmarshal message, terminating: %v", workflowWorkerID, err))
// 		msg.Term() // don't redeliver a poison pill message.
// 		return
// 	}

// 	slog.Debug(fmt.Sprintf("Worker %v received task for Workflow ID: %s", workflowWorkerID, task.WorkflowID))

// 	if task.WorkflowFn == "" {
// 		log.Println("empty workflow Fn name, terminating MSG")
// 		msg.Term()
// 		return
// 	}

// 	// load replay history
// 	history, _, err := i.loadWorkflowHistory(ctx, task.WorkflowID)
// 	if err != nil {
// 		slog.Debug(fmt.Sprintf("ERROR: cannot load history for %s: %v. NAKing task.", task.WorkflowID, err))
// 		msg.Nak()
// 		return
// 	}

// 	// panicked is false only when Workflow Execution is completed (no Activities left to schedule)
// 	panicked := true
// 	// pending is whether the Workflow Execution is actually pending on an un-finished Activity
// 	pending := false
// 	// result is the final return value of the Workflow Function's Business Logic
// 	var results []reflect.Value

// 	// Workflow Execution
// 	wfCtx := newContext(ctx, task.WorkflowFn, task.WorkflowID, history, i.converter)
// 	var newEvents []api.WorkflowEvent
// 	func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				slog.Debug(fmt.Sprintf("PANIC in event handler: %v", r))

// 				if _, ok := r.(ErrorBlockingFuture); !ok {

// 					if err := msg.Nak(); err != nil {
// 						slog.Debug(fmt.Sprintf("Failed to nak message after panic: %v", err))
// 					}
// 				} else {
// 					log.Println("PANIC: is pending")
// 					pending = true
// 				}

// 			} else {
// 				panicked = false
// 			}

// 			// extract the new Workflow History after the call stack is done
// 			newEvents = wfCtx.getNewEvents()
// 		}()

// 		fn, err := i.workflowRegistry.Get(task.WorkflowFn)
// 		if err != nil {
// 			slog.Debug(fmt.Sprintf("no key: %v", err))
// 			return
// 		}

// 		wfv := reflect.ValueOf(fn)
// 		wft := wfv.Type()

// 		// Convert input parameters to match function signature
// 		inputv := make([]reflect.Value, len(task.Input))
// 		slog.Debug(fmt.Sprintf("input params: %v with length: %v", task.Input, len(task.Input)))
// 		for idx, arg := range task.Input {
// 			// Skip the first parameter which is the context
// 			paramType := wft.In(idx + 1)
// 			convertedArg, err := i.convertToType(arg, paramType)
// 			if err != nil {
// 				slog.Debug(fmt.Sprintf("Failed to convert parameter %d: %v", idx, err))
// 				return
// 			}
// 			inputv[idx] = convertedArg
// 		}

// 		results = wfv.Call(append([]reflect.Value{reflect.ValueOf(wfCtx)}, inputv...))

// 	}()

// 	if panicked { // if panicked, either the Workflow Execution is a pending one, which needs Activity Execution
// 		// or the Workflow Execution is actually panic from client code (Workflow Builder) business logic
// 		if pending {

// 			slog.Debug(fmt.Sprintf("pending with %d new events", len(newEvents)))

// 		} else {
// 			slog.Debug(fmt.Sprintf("Workflow %s has failed.", task.WorkflowID))

// 			attrs, _ := i.converter.SerializeBinary(api.WorkflowFailedAttributes{Error: err})
// 			newEvents = append(newEvents, api.WorkflowEvent{
// 				EventType:  api.WorkflowFailed,
// 				WorkflowID: wfCtx.workflowID,
// 				Attributes: attrs,
// 			})
// 		}

// 	} else { // the Workflow Execution is a complete one, which we will need to append an WorkflowCompletedEvent to the new events list
// 		attrs, _ := i.converter.SerializeBinary(api.WorkflowCompletedAttributes{Result: utils.ReflectValuesToAny(results)})
// 		newEvents = append(newEvents, api.WorkflowEvent{
// 			EventType:  api.WorkflowCompleted,
// 			WorkflowID: wfCtx.workflowID,
// 			Attributes: attrs,
// 		})

// 		slog.Debug(fmt.Sprintf("WORKFLOW_COMPLETION_DEBUG: WorkflowCompleted event created and added to newEvents"))
// 	}

// 	// commit changes
// 	slog.Debug(fmt.Sprintf("committing %d new events: %s for workflow %s", len(newEvents), utils.DebugWorkflowEvents(newEvents), task.WorkflowID))
// 	// err = i.commitEvents(ctx, task.WorkflowID, history, newEvents)
// 	err = i.workflowLog.CommitEvents(ctx, task.WorkflowID, history, newEvents)
// 	if err != nil {
// 		slog.Debug(fmt.Sprintf("failed to commit: %v", err))
// 		msg.Nak()
// 		return
// 	}

// 	if err := msg.Ack(); err != nil {
// 		slog.Debug(fmt.Sprintf("Worker %v failed to ACK message: %v", workflowWorkerID, err))
// 	}
// 	slog.Debug(fmt.Sprintf("Workflow Worker %v is finished for now and Acked task for Workflow ID: %s", workflowWorkerID, task.WorkflowID))
// })

// <-ctx.Done()
// wc.Stop()
// log.Println("Workflow task processor has stopped.")

// func (i *workerImpl) runActivityTaskProcessor(ctx context.Context) error {

// 	js, err := i.conn.JS()
// 	if err != nil {
// 		return fmt.Errorf("js not available")
// 	}

// 	activityTaskConsumer, err := js.CreateOrUpdateConsumer(ctx, api.ActivityTasksStream, jetstream.ConsumerConfig{
// 		Durable:       api.ActivityTaskWorkerConsumer,
// 		AckPolicy:     jetstream.AckExplicitPolicy,
// 		FilterSubject: api.ActivityTasksFilterSubjectPattern,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("error: %v", err)
// 	}

// 	cc, err := activityTaskConsumer.Consume(
// 		func(msg jetstream.Msg) {
// 			slog.Debug(fmt.Sprintf("activityTaskConsumer: got msg"))
// 			activityWorkerID := uuid.Must(uuid.NewV7())
// 			if err != nil {
// 				log.Fatalf("Worker %v failed to create consumer: %v", activityWorkerID, err)
// 			}

// 			var task api.ActivityTask
// 			if err := json.Unmarshal(msg.Data(), &task); err != nil {
// 				slog.Debug(fmt.Sprintf("Worker %v could not unmarshal message, terminating: %v", activityWorkerID, err))
// 				msg.Term() // don't redeliver a poison pill message.
// 				return
// 			}
// 			slog.Debug(fmt.Sprintf("activityTaskConsumer: got task %v", task))

// 			fn, err := i.activityRegistry.Get(task.ActivityFn)
// 			if err != nil {
// 				slog.Debug(fmt.Sprintf("activityTaskConsumer: not found in registry: %v", task.ActivityFn))

// 				msg.Nak()
// 				return
// 			}

// 			slog.Debug(fmt.Sprintf("activityTaskConsumer: start executing activity: %v", task.ActivityFn))

// 			result, err := i.executeActivityFunc(ctx, fn, task.Input)

// 			slog.Debug(fmt.Sprintf("activityTaskConsumer: done executing activity: %v", task.ActivityFn))

// 			var resultEvent api.WorkflowEvent
// 			if err != nil {
// 				// The activity returned an error.
// 				slog.Debug(fmt.Sprintf("Activity '%s' for workflow %s failed: %v", task.ActivityFn, task.WorkflowID, err))
// 				resultEvent = &api.ActivityFailedEvent{
// 					Attributes: i.converter.MustTo(api.ActivityFailedAttributes{
// 						Error: err.Error(),
// 					}),
// 				}
// 			} else {
// 				// The activity succeeded.
// 				slog.Debug(fmt.Sprintf("Activity '%s' for workflow %s - %s completed successfully.", task.ActivityFn, task.WorkflowID, task.WorkflowFn))
// 				slog.Debug(fmt.Sprintf("Result of Activity: %s", utils.DebugReflectValues(result)))
// 				// Convert reflect.Value to interface{} before marshaling
// 				resultValues := utils.ReflectValuesToAny(result)
// 				resultData, _ := json.Marshal(resultValues)
// 				resultEvent = &api.ActivityCompletedEvent{
// 					WorkflowID: task.WorkflowID,
// 					Attributes: i.converter.MustTo(api.ActivityCompletedAttributes{
// 						WorkflowFnName: task.WorkflowFn,
// 						ActivityFn:     task.ActivityFn,
// 						Result:         resultData,
// 					}),
// 				}
// 			}

// 			_, err = js.PublishMsg(
// 				ctx,
// 				&nats.Msg{
// 					Subject: fmt.Sprintf(api.HistoryPublishSubjectPattern, task.WorkflowID),
// 					Data:    i.converter.MustTo(resultEvent),
// 				},
// 				jetstream.WithMsgID(activityWorkerID.String()),
// 			)

// 			if err != nil {
// 				// This is a critical failure. We couldn't report the result.
// 				// We MUST NAK the message so another worker can retry the entire activity.
// 				slog.Debug(fmt.Sprintf("CRITICAL: Failed to publish activity result for %s: %v. NAKing task.", task.WorkflowID, err))
// 				msg.Nak()
// 				return
// 			}

// 			slog.Debug(fmt.Sprintf("Sent activity %s event to history.%s", activityWorkerID, task.WorkflowID))

// 			if err = msg.Ack(); err != nil {
// 				slog.Debug(fmt.Sprintf("activity worker failed to Ack: %v", err))
// 				return
// 			}

// 			slog.Debug(fmt.Sprintf("%v", "Done acking"))

// 		})

// 	<-ctx.Done()
// 	cc.Stop()
// 	log.Println("Activity task processor has stopped.")
// 	return nil

// }

// func (i *workerImpl) loadWorkflowHistory(ctx context.Context, workflowID string) ([]api.WorkflowEvent, uint64, error) {
// 	js, _ := i.conn.JS()
// 	history := []api.WorkflowEvent{}
// 	var lastSeq uint64 = 0

// 	stream, err := js.Stream(ctx, "HISTORY")
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	filter := fmt.Sprintf(api.HistoryPublishSubjectPattern, workflowID)

// 	historyConsumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
// 		FilterSubject: filter,
// 		DeliverPolicy: jetstream.DeliverAllPolicy,
// 	})

// 	if err != nil {
// 		slog.Debug(fmt.Sprintf("err: %v", err))

// 		return nil, 0, err
// 	}

// 	fetch, err := historyConsumer.Fetch(1000, jetstream.FetchMaxWait(2*time.Second))
// 	if err != nil {
// 		slog.Debug(fmt.Sprintf("err: %v", err))

// 		return nil, 0, err
// 	}

// 	for msg := range fetch.Messages() {
// 		var event api.WorkflowEvent
// 		meta, err := msg.Metadata()
// 		if err != nil {
// 			slog.Debug(fmt.Sprintf("PROJECTOR/WF: could not get event metadata, terminating: %v", err))
// 			return nil, 0, err
// 		}

// 		if err := json.Unmarshal(msg.Data(), &event); err == nil {
// 			history = append(history, event)
// 			lastSeq = meta.Sequence.Stream
// 		}
// 		msg.Ack()
// 	}

// 	slog.Debug(fmt.Sprintf("getting replay history: %v, lastSeq: %v", utils.DebugWorkflowEvents(history), lastSeq))

// 	return history, lastSeq, nil

// }

// The `lastKnownSeq` parameter is now `lastKnownVersion`.
// func (i *workerImpl) commitEvents(ctx context.Context, workflowID string, history []api.WorkflowEvent, newEvents []api.WorkflowEvent) error {
// 	js, _ := i.conn.JS()
// 	subject := fmt.Sprintf(api.HistoryPublishSubjectPattern, workflowID)

// 	// The last known version of *this specific workflow* is the number of events we loaded for it.
// 	lastKnownVersion := uint64(len(history))
// 	slog.Debug(fmt.Sprintf("last known version: %d", lastKnownVersion))
// 	slog.Debug(fmt.Sprintf("new events: %v", utils.DebugWorkflowEvents(newEvents)))

// 	// If this is the very first event for a new workflow, the history will be empty.
// 	// We must tell Jetstream that we expect NO messages on this subject yet.
// 	if lastKnownVersion == 0 && len(newEvents) > 0 {
// 		eventData := i.converter.MustTo(newEvents[0])

// 		slog.Debug(fmt.Sprintf("committing events to: %v", subject))
// 		_, err := js.PublishMsg(ctx, &nats.Msg{
// 			Subject: subject,
// 			Data:    eventData,
// 		})

// 		if err != nil {
// 			slog.Debug(fmt.Sprintf("err: %v", err))
// 			return err
// 		}

// 		newEvents = newEvents[1:]
// 		lastKnownVersion++
// 	}

// 	// For all subsequent events...
// 	for _, event := range newEvents {
// 		eventData := i.converter.MustTo(event)

// 		expectedVersion := lastKnownVersion

// 		slog.Debug(fmt.Sprintf("committing events to: %v, lastSeq: %v", subject, expectedVersion))

// 		_, err := js.PublishMsg(ctx, &nats.Msg{
// 			Subject: subject,
// 			Data:    eventData,
// 		})

// 		if err != nil {
// 			// This will correctly return ErrWrongLastSequence if another process
// 			// has appended an event to THIS workflow's history in the meantime.
// 			// It is completely unaffected by what other workflows are doing.
// 			slog.Debug(fmt.Sprintf("commit error: %v", err))
// 			return err
// 		}

// 		slog.Debug(fmt.Sprintf("committed event: %v", utils.DebugWorkflowEvents([]api.WorkflowEvent{event})))

// 		lastKnownVersion++
// 	}
// 	return nil
// }

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

	return rawResults, err
}

// convertToType converts a value to the target type, handling JSON unmarshaling quirks
func (w *workerImpl) convertToType(value any, targetType reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(targetType), nil
	}

	valueType := reflect.TypeOf(value)

	// If types already match, return as-is
	if valueType == targetType {
		return reflect.ValueOf(value), nil
	}

	// Handle numeric conversions (JSON unmarshaling converts all numbers to float64)
	if valueType.Kind() == reflect.Float64 && targetType.Kind() == reflect.Int {
		floatVal := value.(float64)
		// Check if it's actually an integer value
		if floatVal == float64(int(floatVal)) {
			return reflect.ValueOf(int(floatVal)), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot convert float64 %v to int without losing precision", floatVal)
	}

	// Handle map[string]interface{} to struct conversion via JSON marshaling/unmarshaling
	if valueType.Kind() == reflect.Map && targetType.Kind() == reflect.Struct {
		// Marshal the map to JSON, then unmarshal to target struct
		jsonData, err := json.Marshal(value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to marshal map to JSON: %v", err)
		}

		// Create a new instance of the target type
		targetValue := reflect.New(targetType).Interface()

		if err := json.Unmarshal(jsonData, targetValue); err != nil {
			return reflect.Value{}, fmt.Errorf("failed to unmarshal JSON to target type: %v", err)
		}

		// Return the dereferenced value (not the pointer)
		return reflect.ValueOf(targetValue).Elem(), nil
	}

	// Handle other numeric conversions if needed
	if valueType.ConvertibleTo(targetType) {
		return reflect.ValueOf(value).Convert(targetType), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %v (%v) to %v", value, valueType, targetType)
}
