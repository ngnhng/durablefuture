package workflow

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/ngnhng/durablefuture/api/serde"
)

// ResultWatcher is the dependency that abstracts the mechanism
// for waiting for a workflow's result.
type ResultWatcher interface {
	// WatchForResult returns a channel that will yield the raw binary result
	// data once the workflow execution is complete and the result is stored.
	// The caller is responsible for deserialization and error checking.
	WatchForResult(ctx context.Context, workflowID string) (<-chan []byte, error)
}

var _ Future = (*Execution)(nil)

type Execution struct {
	WorkflowID string
	watcher    ResultWatcher
	converter  serde.BinarySerde
}

func NewExecution(watcher ResultWatcher, conv serde.BinarySerde, workflowID uuid.UUID) *Execution {
	if conv == nil {
		return &Execution{
			WorkflowID: workflowID.String(),
			watcher:    watcher,
			converter:  conv,
		}
	}
	return &Execution{
		WorkflowID: workflowID.String(),
		watcher:    watcher,
		converter:  conv,
	}
}

func (w *Execution) Get(ctx context.Context, valuePtr any) error {
	// 1. Get the channel from the injected dependency.
	updateChan, err := w.watcher.WatchForResult(ctx, w.WorkflowID)
	if err != nil {
		return fmt.Errorf("error starting result watch: %w", err)
	}

	select {
	case <-ctx.Done():
		// The context was canceled (e.g., timeout).
		return ctx.Err()

	case resultBytes, ok := <-updateChan:
		if !ok {
			// Channel closed unexpectedly without a result.
			return fmt.Errorf("result watcher channel closed unexpectedly")
		}

		// Predefined convention: The workflow result contains [result, error] array
		// TODO: re-consider design choices
		var resultArray []any
		if err := w.converter.DeserializeBinary(resultBytes, &resultArray); err != nil {
			return fmt.Errorf("result deserialization failed: %w", err)
		}

		if len(resultArray) != 2 {
			return fmt.Errorf("invalid workflow result format: expected [result, error], got %d elements", len(resultArray))
		}

		// Extract the error part (second element)
		if resultArray[1] != nil {
			if errStr, ok := resultArray[1].(string); ok && errStr != "" {
				// TODO: errors.Is friendly
				return fmt.Errorf("workflow execution failed: %s", errStr)
			}
			return fmt.Errorf("sdk error: %v", resultArray[1])
		}

		// Process the successful result
		if valuePtr != nil && resultArray[0] != nil {
			// Re-serialize and deserialize to correctly map the result to the
			// target Go type (valuePtr) using the serde implementation.
			resultBytes, _ := w.converter.SerializeBinary(resultArray[0])
			if err := w.converter.DeserializeBinary(resultBytes, valuePtr); err != nil {
				return fmt.Errorf("final result type casting failed: %w", err)
			}
		}

		return nil
	}
}
