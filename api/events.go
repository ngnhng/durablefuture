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

var _ WorkflowEvent = (*WorkflowStarted)(nil)
var _ WorkflowEvent = (*ActivityScheduled)(nil)
var _ WorkflowEvent = (*ActivityStarted)(nil)
var _ WorkflowEvent = (*ActivityCompleted)(nil)
var _ WorkflowEvent = (*ActivityFailed)(nil)
var _ WorkflowEvent = (*WorkflowFailed)(nil)
var _ WorkflowEvent = (*WorkflowCompleted)(nil)

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

	WorkflowFnName string `json:"wf_name"`
	ActivityFnName string `json:"name"`
	Input          []any  `json:"input"`
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
