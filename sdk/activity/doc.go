// Package activity provides types and utilities for writing activities.
//
// Activities are functions that perform non-deterministic operations such as:
//   - Database queries and updates
//   - External API calls
//   - File I/O operations
//   - Any operation with side effects
//
// # Writing Activities
//
// An activity is a regular Go function:
//
//	func MyActivity(ctx context.Context, input string) (string, error) {
//		// Activity logic here - can do I/O, API calls, etc.
//		result, err := externalAPI.Call(input)
//		if err != nil {
//			return "", err
//		}
//		return result, nil
//	}
//
// Activities should accept context.Context as their first parameter to support
// cancellation and timeouts.
//
// # Activity Registration
//
// Activities must be registered with a worker before they can be executed:
//
//	worker.RegisterActivity(MyActivity)
//
// # Calling Activities from Workflows
//
// Activities are called from workflows using workflow.ExecuteActivity:
//
//	var result string
//	err := workflow.ExecuteActivity(ctx, MyActivity, "input").Get(ctx, &result)
//
// # Activity Context
//
// The context.Context passed to an activity contains:
//   - Timeout information
//   - Cancellation signals
//   - Activity metadata (workflow ID, activity ID, attempt number)
//
// # Error Handling
//
// Activities can return errors. Based on the retry policy, the activity may be
// retried automatically. To prevent retries for specific errors, include them
// in the NonRetryableErrorTypes list in the retry policy.
//
// # Best Practices
//
//   - Keep activities focused and single-purpose
//   - Make activities idempotent when possible
//   - Handle errors gracefully
//   - Respect context cancellation
//   - Use appropriate timeouts
package activity
