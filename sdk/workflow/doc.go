// Package workflow provides the programming model for writing durable workflows.
//
// Workflows are deterministic functions that orchestrate activities and handle
// long-running business processes. The workflow package provides the context,
// primitives, and utilities needed to write workflows.
//
// # Writing Workflows
//
// A workflow is a regular Go function that takes a workflow.Context as its first parameter:
//
//	func MyWorkflow(ctx workflow.Context, name string) (string, error) {
//		// Workflow logic here
//		var result string
//		err := workflow.ExecuteActivity(ctx, MyActivity, name).Get(ctx, &result)
//		if err != nil {
//			return "", err
//		}
//		return result, nil
//	}
//
// # Determinism
//
// Workflows must be deterministic. This means:
//   - No direct I/O operations (filesystem, network, database)
//   - No random number generation
//   - No direct time/date operations (use workflow timers)
//   - No access to external mutable state
//   - No goroutines (use workflow.Go instead)
//
// All non-deterministic operations must be performed in activities.
//
// # Activity Execution
//
// Activities are executed using workflow.ExecuteActivity:
//
//	future := workflow.ExecuteActivity(ctx, MyActivity, "arg1", "arg2")
//	var result string
//	if err := future.Get(ctx, &result); err != nil {
//		return "", err
//	}
//
// # Activity Options and Retries
//
// You can configure activity execution with options including retry policies:
//
//	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
//		ScheduleToCloseTimeout: 5 * time.Minute,
//		StartToCloseTimeout:    30 * time.Second,
//		RetryPolicy: &workflow.RetryPolicy{
//			InitialInterval:    time.Second,
//			BackoffCoefficient: 2.0,
//			MaximumInterval:    30 * time.Second,
//			MaximumAttempts:    3,
//			NonRetryableErrorTypes: []string{
//				"invalid input",
//			},
//		},
//	})
//
//	err := workflow.ExecuteActivity(activityCtx, MyActivity, input).Get(ctx, &result)
//
// # Futures
//
// ExecuteActivity returns a Future that represents the asynchronous result.
// You can Get() the result immediately (blocking) or store the Future and
// retrieve it later, enabling parallel execution of multiple activities.
//
// # Context Values
//
// You can pass values through the workflow context:
//
//	ctx = ctx.WithValue("key", "value")
//	value := ctx.Value("key")
//
// # Error Handling
//
// Workflows can return errors. If a workflow returns an error, the workflow
// execution fails and the error is returned to the client.
package workflow
