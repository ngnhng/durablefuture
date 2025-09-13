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

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"durablefuture/internal/converter"
	"durablefuture/internal/types"
	"durablefuture/internal/utils"
	"durablefuture/workflow"
)

type contextImpl struct {
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

var _ workflow.Context = (*contextImpl)(nil)

// futureImpl is the internal implementation.
type futureImpl struct {
	isResolved bool
	value      []byte
	err        error
}

var _ workflow.Future = (*futureImpl)(nil)

func (f *futureImpl) Get(ctx context.Context, resultPtr any) error {
	if !f.isResolved {
		panic(workflow.BlockingFutureError{})
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

func newContext(
	ctx context.Context,
	workflowFn string,
	workflowID string,
	history []types.WorkflowEvent,
	conv converter.Converter) *contextImpl {
	if workflowID == "" {
		log.Println("empty workflow ID")
		panic(fmt.Errorf("empty workflowID"))
	}
	return &contextImpl{
		ctx:                  ctx,
		workflowID:           workflowID,
		workflowFunctionName: workflowFn,
		history:              history,
		newEvents:            make([]types.WorkflowEvent, 0),
		converter:            conv,
	}
}

func (wfCtx *contextImpl) ExecuteActivity(activityFn any, args ...any) workflow.Future {
	return wfCtx.ExecuteActivityWithOptions(activityFn, nil, args...)
}

func (wfCtx *contextImpl) ExecuteActivityWithOptions(activityFn any, options *types.ActivityOptions, args ...any) workflow.Future {
	wfCtx.sequence++
	decisionOutcomesFound := 0
	attemptNumber := 1
	log.Printf("ExecuteActivity: wfCtx seq: %v", wfCtx.sequence)
	log.Printf("ExecuteActivity: wfCtx hist: %v", utils.DebugWorkflowEvents(wfCtx.history))

	// Count the decision outcomes and track retry attempts for this activity
	for _, event := range wfCtx.history {
		if event.EventType == types.ActivityCompletedEvent || event.EventType == types.ActivityFailedEvent {
			decisionOutcomesFound++

			if decisionOutcomesFound == int(wfCtx.sequence) {
				if event.EventType == types.ActivityCompletedEvent {
					log.Printf("[context Execute Activity]: replaying ActivityCompletedEvent: %v", utils.DebugWorkflowEvents([]types.WorkflowEvent{event}))

					var attrs types.ActivityCompletedAttributes
					_ = json.Unmarshal(event.Attributes, &attrs)
					return &futureImpl{isResolved: true, value: attrs.Result}
				} else {
					log.Printf("[context Execute Activity]: replaying ActivityFailedEvent: %v", event)

					var attrs types.ActivityFailedAttributes
					_ = json.Unmarshal(event.Attributes, &attrs)
					
					// Check if we should retry this activity
					if options == nil {
						options = types.DefaultActivityOptions()
					}
					
					if options.RetryPolicy != nil && options.RetryPolicy.ShouldRetry(attrs.AttemptNumber, fmt.Errorf("%s", attrs.Error)) {
						// This failure should trigger a retry - look for a retry scheduled event
						retryFound := false
						for j := len(wfCtx.history) - 1; j >= 0; j-- {
							if wfCtx.history[j].EventType == types.ActivityRetryScheduledEvent {
								var retryAttrs types.ActivityRetryScheduledAttributes
								_ = json.Unmarshal(wfCtx.history[j].Attributes, &retryAttrs)
								if retryAttrs.AttemptNumber == attrs.AttemptNumber + 1 {
									retryFound = true
									attemptNumber = retryAttrs.AttemptNumber
									break
								}
							}
						}
						
						if !retryFound {
							// Schedule a retry
							retryDelay := options.RetryPolicy.CalculateRetryDelay(attrs.AttemptNumber)
							retryEvent := types.WorkflowEvent{
								EventType:  types.ActivityRetryScheduledEvent,
								WorkflowID: wfCtx.GetID(),
								Attributes: wfCtx.converter.MustTo(types.ActivityRetryScheduledAttributes{
									ActivityFnName: "",  // Will be set below
									AttemptNumber:  attrs.AttemptNumber + 1,
									RetryDelay:     retryDelay,
									ScheduledTime:  time.Now().Add(retryDelay),
									Error:          attrs.Error,
								}),
							}
							wfCtx.newEvents = append(wfCtx.newEvents, retryEvent)
							attemptNumber = attrs.AttemptNumber + 1
						}
						
						// Return an unresolved future to trigger re-execution
						return &futureImpl{isResolved: false}
					}
					
					// No retry - return the error
					return &futureImpl{isResolved: true, err: fmt.Errorf("%s", attrs.Error)}
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

	log.Printf("[context] will try to execute activity: %v, attempt: %v", fnName, attemptNumber)

	// Use default options if none provided
	if options == nil {
		options = types.DefaultActivityOptions()
	}

	attributes, err := json.Marshal(types.ActivityTaskScheduledAttributes{
		WorkflowFnName:  wfCtx.GetWorkflowFunctionName(),
		ActivityFnName:  fnName,
		Input:           args,
		ActivityOptions: options,
		AttemptNumber:   attemptNumber,
		ScheduledTime:   time.Now(),
	})
	if err != nil {
		log.Printf("error: %v", err)
		panic("marshal error")
	}

	log.Printf("new activity task scheduled event for: %v with ID: %v, attempt: %v", fnName, wfCtx.GetID(), attemptNumber)
	newEvent := types.WorkflowEvent{
		EventType:  types.ActivityTaskScheduledEvent,
		WorkflowID: wfCtx.GetID(),
		Attributes: attributes,
	}
	wfCtx.newEvents = append(wfCtx.newEvents, newEvent)

	log.Printf("ExecuteActivity: new added events: %v", utils.DebugWorkflowEvents(wfCtx.newEvents))

	return &futureImpl{isResolved: false}
}

// getNewEvents is an unexported method accessible only within the `workflow` package.
// The executor calls this to retrieve the commands generated by the workflow logic.
func (c *contextImpl) getNewEvents() []types.WorkflowEvent {
	return c.newEvents
}

func (c *contextImpl) GetID() string {
	return c.workflowID
}

func (c *contextImpl) GetWorkflowFunctionName() string {
	return c.workflowFunctionName
}

func (c *contextImpl) Deadline() (deadline time.Time, ok bool) { return c.ctx.Deadline() }
func (c *contextImpl) Done() <-chan struct{}                   { return c.ctx.Done() }
func (c *contextImpl) Err() error                              { return c.ctx.Err() }
func (c *contextImpl) Value(key any) any                       { return c.ctx.Value(key) }
