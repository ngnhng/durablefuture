package api

type WorkflowCommandType string

const (
	StartWorkflowCommand WorkflowCommandType = "StartWorkflow"
)

type (
	Command struct {
		CommandType WorkflowCommandType `json:"type"`
		Attributes  []byte              `json:"attributes"`
	}

	StartWorkflowAttributes struct {
		WorkflowFnName string `json:"workflow_fn_name"`
		Input          []any  `json:"input"`
	}

	StartWorkflowReply struct {
		Error      string `json:"error,omitempty"`
		WorkflowID string `json:"workflow_id"`
	}
)

type (
	Task interface {
		isTask()
	}

	WorkflowTask struct {
		WorkflowID string `json:"wf_id"`
		WorkflowFn string `json:"wf_name"`
		Input      []any  `json:"input"`
	}

	ActivityTask struct {
		WorkflowFn                 string       `json:"wf_name"`
		WorkflowID                 string       `json:"wf_id"`
		ActivityFn                 string       `json:"ac_name"`
		Input                      []any        `json:"input"`
		LastSeq                    int          `json:"last_seq"`
		Attempt                    int32        `json:"attempt"`
		ScheduleToCloseTimeoutUnix int64        `json:"schedule_to_close_timeout_ms,omitempty"`
		StartToCloseTimeoutUnix    int64        `json:"start_to_close_timeout_ms,omitempty"`
		RetryPolicy                *RetryPolicy `json:"retry_policy,omitempty"`
		ScheduledAtMs              int64        `json:"scheduled_at_ms,omitempty"` // timestamp when first scheduled
	}
)

func (t *WorkflowTask) isTask() {}
func (t *ActivityTask) isTask() {}

// RetryPolicy defines the retry behavior for activities
type RetryPolicy struct {
	// InitialIntervalMs is the backoff interval for the first retry in milliseconds.
	// If BackoffCoefficient is 1.0 then it is used for all retries.
	// Default is 1000ms (1 second).
	InitialIntervalMs int64 `json:"initial_interval_ms,omitempty"`

	// BackoffCoefficient is used to calculate the next retry backoff interval.
	// The next retry interval is previous interval multiplied by this coefficient.
	// Must be 1 or larger. Default is 2.0.
	BackoffCoefficient float64 `json:"backoff_coefficient,omitempty"`

	// MaximumIntervalMs is the maximum backoff interval between retries in milliseconds.
	// Exponential backoff leads to interval increase. This value caps the interval.
	// Default is 100x of initial interval.
	MaximumIntervalMs int64 `json:"maximum_interval_ms,omitempty"`

	// MaximumAttempts is the maximum number of attempts. When exceeded, retries stop.
	// If not set or set to 0, it means unlimited (relies on ScheduleToCloseTimeout).
	MaximumAttempts int32 `json:"maximum_attempts,omitempty"`

	// NonRetryableErrorTypes lists error messages that should not be retried.
	// If an activity fails with an error matching any of these strings, it won't be retried.
	NonRetryableErrorTypes []string `json:"non_retryable_error_types,omitempty"`
}
