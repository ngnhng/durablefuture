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

import "fmt"

// WorkflowInputKey returns the KV key used to persist workflow inputs.
//
// Keys are scoped by the workflow function name and workflow ID so parallel runs don't stomp each other's inputs.
//
// A valid key can contain the following characters: a-z, A-Z, 0-9, _, -, ., = and /, i.e. it can be a dot-separated list of tokens (which means that you can then use wildcards to match hierarchies of keys when watching a bucket). The value can be any byte array.
func WorkflowInputKey(workflowFnName string, workflowID WorkflowID) string {
	return fmt.Sprintf("%s/%s", workflowFnName, workflowID)
}
