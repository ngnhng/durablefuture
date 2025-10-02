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
		WorkflowFn string `json:"wf_name"`
		WorkflowID string `json:"wf_id"`
		ActivityFn string `json:"ac_name"`
		Input      []any  `json:"input"`
		LastSeq    int    `json:"last_seq"`
	}
)

func (t *WorkflowTask) isTask() {}
func (t *ActivityTask) isTask() {}
