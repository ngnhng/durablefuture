package api

import "fmt"

// WorkflowInputKey returns the KV key used to persist workflow inputs.
//
// Keys are scoped by the workflow function name and workflow ID so parallel runs don't stomp each other's inputs.
//
// A valid key can contain the following characters: a-z, A-Z, 0-9, _, -, ., = and /, i.e. it can be a dot-separated list of tokens (which means that you can then use wildcards to match hierarchies of keys when watching a bucket). The value can be any byte array.
func WorkflowInputKey(workflowFnName string, workflowID WorkflowID) string {
	return fmt.Sprintf("%s/%s", workflowFnName, workflowID)
}
