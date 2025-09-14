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

package constants

// NATS Stream Names
const (
	WorkflowHistoryStream = "WORKFLOW_HISTORY"
	WorkflowCommandStream = "WORKFLOW_COMMAND"
	WorkflowTasksStream   = "WORKFLOW_TASKS"
	ActivityTasksStream   = "ACTIVITY_TASKS"
	WorkflowResultStream  = "WORKFLOW_RESULT"
	ActivityResultStream  = "ACTIVITY_RESULT"
	HistoryStream         = "HISTORY"
)

// NATS Subject Prefixes
const (
	HistorySubjectPrefix        = "history"
	CommandSubjectPrefix        = "command"
	WorkflowTasksSubjectPrefix  = "workflow.tasks"
	ActivityTaskSubjectPrefix   = "activity.tasks"
	WorkflowResultSubjectPrefix = "workflow.result"
	ActivityResultSubjectPrefix = "activity.result"
)

// NATS Subject Patterns
const (
	HistorySubjectPattern        = "history.>"
	CommandRequestSubjectPattern = "command.request.>"
	WorkflowTasksSubjectPattern  = "workflow.*.tasks"
	ActivityTasksSubjectPattern  = "activity.*.tasks"
)

// Specific Command Subjects
const (
	CommandRequestStart = "command.request.start"
)

// Consumer Names
const (
	ManagerCommandProcessorsConsumer = "manager-command-processors"
	WorkflowTaskProjectorConsumer    = "projector-workflow-tasks"
	ActivityTaskProjectorConsumer    = "projector-activity-tasks"
	WorkflowResultProjectorConsumer  = "projector-workflow-result"
)

// KeyValue Bucket Names
const (
	WorkflowResultBucket = "workflow-result"
	WorkflowInputBucket  = "workflow-input"
)

// Test and Example Values
const (
	DefaultCarrier        = "FastShip Express"
	TestFailureCustomerID = "fail_customer"
)