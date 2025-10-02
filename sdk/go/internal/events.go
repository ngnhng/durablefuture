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
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.ActivityScheduled:
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.ActivityStarted:
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.ActivityCompleted:
		c.workflowFunctionName = evt.WorkflowFnName
		c.activityResult[evt.ActivityFnName] = evt.Result
	case *api.ActivityFailed:
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.WorkflowFailed:
		c.workflowFunctionName = evt.WorkflowFnName
	case *api.WorkflowCompleted:
		c.workflowFunctionName = evt.WorkflowFnName
	default:
		return fmt.Errorf("unknown event type: %T", e)
	}
	return nil
}
