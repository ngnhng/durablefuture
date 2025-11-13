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

// NATS Stream Names
const (
	WorkflowHistoryStream = "WORKFLOW_HISTORY"
	WorkflowTasksStream   = "WORKFLOW_TASKS"
	ActivityTasksStream   = "ACTIVITY_TASKS"
)

// NATS Subject Prefix
const (
	HistorySubjectPrefix = "history"
)

// NATS Subject Format
const (
	HistoryPublishSubjectPattern = HistorySubjectPrefix + ".%s" // workflowID
)

// NATS Subject Patterns
const (
	HistoryFilterSubjectPattern = HistorySubjectPrefix + ".>"

	CommandRequestSubjectPattern = "command.request.>"

	WorkflowTasksFilterSubjectPattern = "workflow.*.tasks"
	ActivityTasksFilterSubjectPattern = "activity.*.tasks"
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
	ActivityRetryProjectorConsumer   = "projector-activity-retries"
	WorkflowResultProjectorConsumer  = "projector-workflow-result"

	WorkflowTaskWorkerConsumer = "worker-workflow-tasks"
	ActivityTaskWorkerConsumer = "worker-activity-tasks"
)

// KeyValue Bucket Names
const (
	WorkflowResultBucket = "workflow-result"
	WorkflowInputBucket  = "workflow-input"
)

// JetStream Headers
const (
	ChronicleEventNameHeader        = "Chronicle-Event-Name"
	ChronicleAggregateVersionHeader = "Chronicle-Aggregate-Version"
)
