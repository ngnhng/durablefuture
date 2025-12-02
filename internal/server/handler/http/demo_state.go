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
