package http

import (
	"encoding/json"
	"time"
)

// WorkflowState represents the current state of a workflow execution
type WorkflowState struct {
	WorkflowID  string          `json:"workflow_id"`
	Status      string          `json:"status"` // "pending", "running", "completed", "failed"
	CurrentStep int             `json:"current_step"`
	Steps       []string        `json:"steps"`
	StepDetails []StepDetail    `json:"step_details"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// StepDetail represents the status of an individual workflow step (activity)
type StepDetail struct {
	Name         string         `json:"name"`
	Status       string         `json:"status"` // "pending", "running", "completed", "failed"
	Attempts     int            `json:"attempts"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	EndedAt      *time.Time     `json:"ended_at,omitempty"`
	Error        string         `json:"error,omitempty"`
	RetryHistory []RetryAttempt `json:"retry_history,omitempty"`
}

// RetryAttempt represents a single retry attempt for an activity
type RetryAttempt struct {
	Attempt        int       `json:"attempt"`
	Error          string    `json:"error,omitempty"`
	NextRetryDelay int64     `json:"next_retry_delay_ms,omitempty"`
	RecordedAt     time.Time `json:"recorded_at"`
}
