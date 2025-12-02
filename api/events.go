package api

import (
	"github.com/DeluxeOwl/chronicle/event"
)

type WorkflowID string

func (w WorkflowID) String() string { return string(w) }

type WorkflowEvent interface {
	event.Any

	isWorkflowEvent()
}

var (
	_ WorkflowEvent = (*WorkflowStarted)(nil)
	_ WorkflowEvent = (*ActivityScheduled)(nil)
	_ WorkflowEvent = (*ActivityStarted)(nil)
	_ WorkflowEvent = (*ActivityCompleted)(nil)
	_ WorkflowEvent = (*ActivityFailed)(nil)
	_ WorkflowEvent = (*ActivityRetried)(nil)
	_ WorkflowEvent = (*WorkflowFailed)(nil)
	_ WorkflowEvent = (*WorkflowCompleted)(nil)
)

// TODO: workflow cancelled, workflow timed out, workflow terminated

// -- Workflow Started Event --
type WorkflowStarted struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"name"`
	Input          []any  `json:"input"`
}

func (*WorkflowStarted) EventName() string { return "workflow/started" }
func (*WorkflowStarted) isWorkflowEvent()  {}

// -- Activity Scheduled Event --
type ActivityScheduled struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName             string       `json:"wf_name"`
	ActivityFnName             string       `json:"name"`
	Input                      []any        `json:"input"`
	ScheduleToCloseTimeoutUnix int64        `json:"schedule_to_close_timeout_unix,omitempty"`
	StartToCloseTimeoutUnix    int64        `json:"start_to_close_timeout_ms,omitempty"`
	RetryPolicy                *RetryPolicy `json:"retry_policy,omitempty"`
}

func (*ActivityScheduled) EventName() string { return "activity/scheduled" }
func (*ActivityScheduled) isWorkflowEvent()  {}

// -- Activity Started Event --
type ActivityStarted struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"wf_name"`
	ActivityFnName string `json:"name"`
}

func (*ActivityStarted) EventName() string { return "activity/started" }
func (*ActivityStarted) isWorkflowEvent()  {}

// -- Activity Completed Event --
type ActivityCompleted struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"wf_name"`
	ActivityFnName string `json:"name"`
	Result         []any  `json:"result"`
}

func (*ActivityCompleted) EventName() string { return "activity/completed" }
func (*ActivityCompleted) isWorkflowEvent()  {}

// -- Activity Failed Event --
type ActivityFailed struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"wf_name"`
	ActivityFnName string `json:"name"`
	Error          string `json:"error"`
}

func (*ActivityFailed) EventName() string { return "activity/failed" }
func (*ActivityFailed) isWorkflowEvent()  {}

// -- Activity Retried Event --
type ActivityRetried struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"wf_name"`
	ActivityFnName string `json:"name"`
	Attempt        int32  `json:"attempt"`
	Error          string `json:"error"`
	NextRetryDelay int64  `json:"next_retry_delay_ms"` // milliseconds
}

func (*ActivityRetried) EventName() string { return "activity/retried" }
func (*ActivityRetried) isWorkflowEvent()  {}

// -- Workflow Failed --
type WorkflowFailed struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"name"`
	Error          string `json:"error"`
}

func (*WorkflowFailed) EventName() string { return "workflow/failed" }
func (*WorkflowFailed) isWorkflowEvent()  {}

// -- Workflow Completed --
type WorkflowCompleted struct {
	ID WorkflowID `json:"id"`

	WorkflowFnName string `json:"name"`
	Result         []any  `json:"result"`
}

func (*WorkflowCompleted) EventName() string { return "workflow/completed" }
func (*WorkflowCompleted) isWorkflowEvent()  {}
