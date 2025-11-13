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
	"fmt"

	"github.com/DeluxeOwl/chronicle/aggregate"
	"github.com/DeluxeOwl/chronicle/event"
	"github.com/ngnhng/durablefuture/api"
)

func (c *workflowContext) EventFuncs() event.FuncsFor[api.WorkflowEvent] {
	return event.FuncsFor[api.WorkflowEvent]{
		func() api.WorkflowEvent { return new(api.WorkflowStarted) },
		func() api.WorkflowEvent { return new(api.ActivityScheduled) },
		func() api.WorkflowEvent { return new(api.ActivityStarted) },
		func() api.WorkflowEvent { return new(api.ActivityCompleted) },
		func() api.WorkflowEvent { return new(api.ActivityFailed) },
		func() api.WorkflowEvent { return new(api.ActivityRetried) },
		func() api.WorkflowEvent { return new(api.WorkflowFailed) },
		func() api.WorkflowEvent { return new(api.WorkflowCompleted) },
	}
}

func (c *workflowContext) ID() api.WorkflowID { return c.id }

func (c *workflowContext) recordThat(e api.WorkflowEvent) error {
	return aggregate.RecordEvent(c, e)
}

func (c *workflowContext) Apply(e api.WorkflowEvent) error {
	switch evt := e.(type) {
	case *api.WorkflowStarted:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.ActivityScheduled:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.ActivityStarted:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.ActivityCompleted:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
		entry := c.ensureActivityReplay(evt.ActivityFnName)
		entry.history = append(entry.history, activityReplayRecord{
			result: evt.Result,
		})
	case *api.ActivityFailed:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
		entry := c.ensureActivityReplay(evt.ActivityFnName)
		entry.history = append(entry.history, activityReplayRecord{
			err: fmt.Errorf("%s", evt.Error),
		})
	case *api.ActivityRetried:
		// ActivityRetried events don't affect replay state directly
		// They are informational only - the retry projector will reschedule the activity
		// During replay, we skip these events and wait for the final Completed/Failed event
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.WorkflowFailed:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.WorkflowCompleted:
		c.id = evt.ID
		c.workflowFunctionName = evt.WorkflowFnName
	default:
		return fmt.Errorf("unknown event type: %T", e)
	}
	return nil
}
