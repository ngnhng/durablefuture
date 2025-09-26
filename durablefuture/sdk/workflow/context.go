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
	"context"
	"durablefuture/common/converter"
	"durablefuture/common/types"
	"durablefuture/common/utils"
	"durablefuture/sdk/internal/future"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Context is the interface for Workflow Operations.
type Context interface {
	context.Context

	ExecuteActivity(activityFn any, args ...any) Future

	GetID() string

	GetWorkflowFunctionName() string

	WithValue(key any, value any) Context
}

var _ Context = (*ContextImpl)(nil)

// ExecuteActivity schedules the execution of an activity function.
// This is a top-level function that workflow authors will call.
//
// The `ctx` (workflow.Context) holds the replay state.
// The `activityFn` must be a function registered with the worker.
// `args` are the arguments to pass to the activity. They must be JSON-serializable.
func ExecuteActivity(ctx Context, activityFn any, args ...any) Future {
	return ctx.ExecuteActivity(activityFn, args...)
}

type ContextImpl struct {
	// history is the rehydrated history for this Workflow Context
	history []types.WorkflowEvent
	// sequence is the local event sequence up to now in this Workflow Context
	sequence uint64

	newEvents []types.WorkflowEvent

	converter converter.Converter

	workflowID string

	ctx context.Context

	workflowFunctionName string
}

// FutureImpl is the internal implementation.
type FutureImpl struct {
	isResolved bool
	value      []byte
	err        error
}

func (f *FutureImpl) Get(ctx context.Context, resultPtr any) error {
	if !f.isResolved {
		panic(future.BlockingFutureError{})
	}
	if f.err != nil {
		return f.err
	}
	if resultPtr != nil && f.value != nil {

		var resultArray []any
		err := json.Unmarshal(f.value, &resultArray)
		if err != nil {
			return err
		}

		log.Printf("[Activity Get] %v", utils.DebugAnyValues(resultArray))

		// Check if we have both result and error parts
		if len(resultArray) != 2 {
			return fmt.Errorf("invalid workflow result format: expected [result, error], got %d elements", len(resultArray))
		}
		// Extract the error part (second element)
		if resultArray[1] != nil {
			// If error is not nil, return it as the Get method's error
			if errStr, ok := resultArray[1].(string); ok && errStr != "" {
				return fmt.Errorf("%s", errStr)
			}
			// Handle case where error is not a string
			return fmt.Errorf("[Activity Get] workflow execution failed: %v", resultArray[1])
		}

		// No error, unmarshal the result part (first element) into valuePtr
		if resultArray[0] != nil {
			// Convert the result part back to JSON and then unmarshal into valuePtr
			resultJSON, err := json.Marshal(resultArray[0])
			if err != nil {
				return err
			}
			return json.Unmarshal(resultJSON, resultPtr)
		}

	}
	return nil
}

func (wfCtx *ContextImpl) ExecuteActivity(activityFn any, args ...any) Future {
	opts := getActivityOptions(wfCtx)

	wfCtx.sequence++
	decisionOutcomesFound := 0
	log.Printf("ExecuteActivity: wfCtx seq: %v", wfCtx.sequence)
	log.Printf("ExecuteActivity: wfCtx hist: %v", utils.DebugWorkflowEvents(wfCtx.history))

	for _, event := range wfCtx.history {
		if event.EventType == types.ActivityCompletedEvent || event.EventType == types.ActivityFailedEvent {
			decisionOutcomesFound++

			if decisionOutcomesFound == int(wfCtx.sequence) {
				if event.EventType == types.ActivityCompletedEvent {
					log.Printf("[context Execute Activity]: replaying ActivityCompletedEvent: %v", utils.DebugWorkflowEvents([]types.WorkflowEvent{event}))

					var attrs types.ActivityCompletedAttributes
					_ = json.Unmarshal(event.Attributes, &attrs)
					return &FutureImpl{isResolved: true, value: attrs.Result}
				} else {
					log.Printf("[context Execute Activity]: replaying ActivityFailedEvent: %v", event)

					var attrs types.ActivityFailedAttributes
					_ = json.Unmarshal(event.Attributes, &attrs)
					return &FutureImpl{isResolved: true, err: fmt.Errorf(attrs.Error)}
				}
			}
		}
	}

	// --- First-Time Execution Logic ---
	// If we've iterated through the entire history and haven't found the Nth outcome,
	// it means this is a new decision to be made.
	fnName, err := utils.ExtractFullFunctionName(activityFn)
	if err != nil { /* panic */
		log.Printf("error: %v", err)
		panic(err)
	}

	log.Printf("[context] will try to execute activity: %v", fnName)

	attributes, err := json.Marshal(types.ActivityTaskScheduledAttributes{
		WorkflowFnName: wfCtx.GetWorkflowFunctionName(),
		ActivityFnName: fnName,
		Input:          args,
	})
	if err != nil {
		log.Printf("error: %v", err)
		panic("marshal error")
	}

	log.Printf("new activity task scheduled event for: %v with ID: %v", fnName, wfCtx.GetID())
	newEvent := types.WorkflowEvent{
		EventType:  types.ActivityTaskScheduledEvent,
		WorkflowID: wfCtx.GetID(),
		Attributes: attributes,
	}
	wfCtx.newEvents = append(wfCtx.newEvents, newEvent)

	log.Printf("ExecuteActivity: new added events: %v", utils.DebugWorkflowEvents(wfCtx.newEvents))

	return &FutureImpl{isResolved: false}
}

func getActivityOptions(ctx *ContextImpl) ActivityOptions {
	val := ctx.Value(activityOptionsKey{})
	if val == nil {
		log.Panic("ActivityOptions not found in context. Please use WithActivityOptions to set it.")
	}
	opts, ok := val.(ActivityOptions)
	if !ok {
		log.Panic("ActivityOptions has wrong type in context.")
	}
	return opts
}

func (wfCtx *ContextImpl) WithValue(key any, value any) Context {
	return &ContextImpl{
		history:              wfCtx.history,
		sequence:             wfCtx.sequence,
		newEvents:            wfCtx.newEvents,
		converter:            wfCtx.converter,
		workflowID:           wfCtx.workflowID,
		ctx:                  context.WithValue(wfCtx.ctx, key, value),
		workflowFunctionName: wfCtx.workflowFunctionName,
	}
}

// getNewEvents is an unexported method accessible only within the `workflow` package.
// The executor calls this to retrieve the commands generated by the workflow logic.
func (c *ContextImpl) getNewEvents() []types.WorkflowEvent     { return c.newEvents }
func (c *ContextImpl) GetID() string                           { return c.workflowID }
func (c *ContextImpl) GetWorkflowFunctionName() string         { return c.workflowFunctionName }
func (c *ContextImpl) Deadline() (deadline time.Time, ok bool) { return c.ctx.Deadline() }
func (c *ContextImpl) Done() <-chan struct{}                   { return c.ctx.Done() }
func (c *ContextImpl) Err() error                              { return c.ctx.Err() }
func (c *ContextImpl) Value(key any) any                       { return c.ctx.Value(key) }
