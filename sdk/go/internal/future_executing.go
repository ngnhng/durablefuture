package internal

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/ngnhng/durablefuture/api/serde"
)

type WorkflowResultWatcher interface {
	// Watch waits for the result of a given workflow and returns the raw data.
	// It should block until the result is available or the context is canceled.
	Watch(ctx context.Context, workflowID string) ([]byte, error)
}

type WorkflowResult struct {
	Value any
	Err   error
}

var _ Future = (*executing)(nil)

type executing struct {
	c          Client
	watcher    WorkflowResultWatcher
	converter  serde.BinarySerde
	workflowID string
}

func NewExecution(c Client, watcher WorkflowResultWatcher, conv serde.BinarySerde, workflowID uuid.UUID) *executing {
	return &executing{
		c:          c,
		watcher:    watcher,
		converter:  conv,
		workflowID: workflowID.String(),
	}
}

func (e *executing) Get(ctx context.Context, valuePtr any) error {
	// Delegate the waiting and data fetching to the watcher.
	rawData, err := e.watcher.Watch(ctx, e.workflowID)
	if err != nil {
		return fmt.Errorf("error waiting for workflow result: %w", err)
	}

	if rawData == nil {
		// This can happen if the watcher implementation has a bug or the context is canceled
		// immediately.
		return fmt.Errorf("watcher returned no data for workflow %s", e.workflowID)
	}
	// Now, parse the business-level [result, error] format from the raw data.
	var resultArray []any
	if err := e.converter.DeserializeBinary(rawData, &resultArray); err != nil {
		return fmt.Errorf("failed to deserialize workflow result container: %w", err)
	}

	if len(resultArray) != 2 {
		return fmt.Errorf("invalid workflow result format: expected [result, error], got %d elements", len(resultArray))
	}

	// Check for an execution error from the workflow.
	if resultArray[1] != nil {
		if errStr, ok := resultArray[1].(string); ok && errStr != "" {
			return fmt.Errorf("workflow execution failed: %s", errStr)
		}
		return fmt.Errorf("sdk received an unknown error from workflow: %v", resultArray[1])
	}

	// If we are here, the workflow was successful. Deserialize the actual result.
	if valuePtr != nil && resultArray[0] != nil {
		// This is the crucial step to correctly populate the user's struct.
		// We re-serialize the generic 'any' (map[string]any) to bytes
		// and then deserialize into the specific user-provided pointer.
		tempBytes, err := e.converter.SerializeBinary(resultArray[0])
		if err != nil {
			return fmt.Errorf("failed to re-serialize workflow result value: %w", err)
		}
		if err := e.converter.DeserializeBinary(tempBytes, valuePtr); err != nil {
			return fmt.Errorf("failed to deserialize result value into provided type: %w", err)
		}
	}

	return nil

}
