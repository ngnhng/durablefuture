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

package workflow

import (
	"github.com/ngnhng/durablefuture/sdk/internal"
)

// Context is the workflow execution context that provides deterministic guarantees.
//
// Context extends context.Context with workflow-specific operations. All workflow
// operations must go through this context to maintain determinism during replay.
//
// Key methods:
//   - ExecuteActivity: Schedule an activity for execution
//   - WithValue: Store values in the workflow context
//   - Done/Deadline/Err/Value: Standard context.Context methods
//
// Important: Workflow code must be deterministic. Do not:
//   - Perform I/O operations directly
//   - Generate random numbers
//   - Access current time directly
//   - Use goroutines
//
// Use activities for all non-deterministic operations.
type Context = internal.Context

// ExecuteActivity schedules the execution of an activity function.
// This is a top-level function that workflow authors will call.
//
// The `ctx` (workflow.Context) holds the replay state.
// The `activityFn` must be a function registered with the worker.
// `args` are the arguments to pass to the activity. They must be JSON-serializable.
func ExecuteActivity(ctx Context, activityFn any, args ...any) Future {
	return ctx.ExecuteActivity(activityFn, args...)
}
