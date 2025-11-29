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

package http

// findOrCreateStep finds a step by name in the workflow state or creates it if not found
func (h *DemoHandler) findOrCreateStep(state *WorkflowState, activityName string) int {
	for i, step := range state.StepDetails {
		if step.Name == activityName {
			return i
		}
	}

	// Create new step
	state.StepDetails = append(state.StepDetails, StepDetail{
		Name:   activityName,
		Status: "pending",
	})
	return len(state.StepDetails) - 1
}

// findStepByName finds a step by name in the workflow state
func (h *DemoHandler) findStepByName(state *WorkflowState, activityName string) int {
	for i, step := range state.StepDetails {
		if step.Name == activityName {
			return i
		}
	}
	return -1
}
