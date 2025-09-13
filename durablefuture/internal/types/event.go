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

package types

import (
	"encoding/json"
	"time"
)

type (
	WorkflowEvent struct {
		EventID    int64             `json:"id"`
		EventType  WorkflowEventType `json:"type"`
		WorkflowID string            `json:"workflowID"`
		Attributes []byte            `json:"attributes"`
	}
)

type WorkflowEventType int

const (
	WorkflowStartedEvent WorkflowEventType = iota
	ActivityTaskScheduledEvent
	ActivityStarted
	ActivityCompletedEvent
	ActivityFailedEvent
	ActivityRetryScheduledEvent
	ActivityTimedOutEvent
	WorkflowFailed
	WorkflowCompleted
)

type (
	WorkflowStartedAttributes struct {
		WorkflowFnName string `json:"name"`
		Input          []any  `json:"input"`
	}

	WorkflowFailedAttributes struct {
		Error error
	}

	WorkflowCompletedAttributes struct {
		Result []any
	}

	ActivityTaskScheduledAttributes struct {
		WorkflowFnName   string          `json:"wf_name"`
		ActivityFnName   string          `json:"name"`
		Input            []any           `json:"input"`
		ActivityOptions  *ActivityOptions `json:"activity_options,omitempty"`
		AttemptNumber    int             `json:"attempt_number"`
		ScheduledTime    time.Time       `json:"scheduled_time"`
	}

	ActivityCompletedAttributes struct {
		WorkflowFnName string          `json:"wf_name"`
		ActivityFn     string          `json:"name"`
		Result         json.RawMessage `json:"result"`
	}

	ActivityFailedAttributes struct {
		Error         string `json:"error"`
		AttemptNumber int    `json:"attempt_number"`
		FailedTime    time.Time `json:"failed_time"`
	}

	ActivityRetryScheduledAttributes struct {
		ActivityFnName string          `json:"name"`
		AttemptNumber  int             `json:"attempt_number"`
		RetryDelay     time.Duration   `json:"retry_delay"`
		ScheduledTime  time.Time       `json:"scheduled_time"`
		Error          string          `json:"error"`
	}

	ActivityTimedOutAttributes struct {
		ActivityFnName string        `json:"name"`
		AttemptNumber  int           `json:"attempt_number"`
		TimeoutType    string        `json:"timeout_type"` // "ScheduleToStart", "StartToClose", "Heartbeat"
		TimedOutTime   time.Time     `json:"timed_out_time"`
	}
)
