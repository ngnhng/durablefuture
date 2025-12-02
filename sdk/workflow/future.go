package workflow

import (
	"github.com/ngnhng/durablefuture/sdk/internal"
)

// Future represents the result of an asynchronous operation (activity or child workflow).
//
// A Future is returned by workflow.ExecuteActivity and provides methods to retrieve
// the result. Futures enable parallel execution by allowing you to start multiple
// activities and then wait for their results later.
//
// Example usage:
//
//	// Start multiple activities in parallel
//	future1 := workflow.ExecuteActivity(ctx, Activity1, arg1)
//	future2 := workflow.ExecuteActivity(ctx, Activity2, arg2)
//
//	// Wait for results
//	var result1 string
//	if err := future1.Get(ctx, &result1); err != nil {
//		return err
//	}
//
//	var result2 int
//	if err := future2.Get(ctx, &result2); err != nil {
//		return err
//	}
//
// The Get method blocks until the operation completes. During workflow replay,
// Get returns cached results immediately without blocking.
type Future = internal.Future
