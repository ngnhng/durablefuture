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

package workflow

import (
	"context"
	"durablefuture/internal/types"
)

// Context is the interface for Workflow Operations.
type Context interface {
	context.Context

	ExecuteActivity(activityFn any, args ...any) Future
	ExecuteActivityWithOptions(activityFn any, options *types.ActivityOptions, args ...any) Future

	GetID() string

	GetWorkflowFunctionName() string
}

// ExecuteActivity schedules the execution of an activity function.
// This is a top-level function that workflow authors will call.
//
// The `ctx` (workflow.Context) holds the replay state.
// The `activityFn` must be a function registered with the worker.
// `args` are the arguments to pass to the activity. They must be JSON-serializable.
func ExecuteActivity(ctx Context, activityFn any, args ...any) Future {
	return ctx.ExecuteActivity(activityFn, args...)
}

// ExecuteActivityWithOptions schedules the execution of an activity function with custom options.
// This allows configuring retry policies, timeouts, and other execution parameters.
//
// The `ctx` (workflow.Context) holds the replay state.
// The `activityFn` must be a function registered with the worker.
// The `options` specify custom execution parameters like retry policy and timeouts.
// `args` are the arguments to pass to the activity. They must be JSON-serializable.
func ExecuteActivityWithOptions(ctx Context, activityFn any, options *types.ActivityOptions, args ...any) Future {
	return ctx.ExecuteActivityWithOptions(activityFn, options, args...)
}
