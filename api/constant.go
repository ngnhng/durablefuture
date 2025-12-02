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
