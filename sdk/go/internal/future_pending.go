package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ngnhng/durablefuture/server/utils"
)

var _ Future = (*pending)(nil)

type Future interface {
	Get(ctx context.Context, valuePtr any) error
}

// pending is the internal implementation.
type pending struct {
	isResolved bool
	value      []any
	err        error
}

func (f *pending) Get(ctx context.Context, resultPtr any) error {
	if !f.isResolved {
		panic(ErrorBlockingFuture{})
	}
	if f.err != nil {
		return f.err
	}
	if resultPtr != nil && f.value != nil {

		log.Printf("[Activity Get] %v", utils.DebugAnyValues(f.value))

		// Check if we have both result and error parts
		if len(f.value) != 2 {
			return fmt.Errorf("invalid workflow result format: expected [result, error], got %d elements", len(f.value))
		}
		// Extract the error part (second element)
		if f.value[1] != nil {
			// If error is not nil, return it as the Get method's error
			if errStr, ok := f.value[1].(string); ok && errStr != "" {
				return fmt.Errorf("%s", errStr)
			}
			// Handle case where error is not a string
			return fmt.Errorf("[Activity Get] workflow execution failed: %v", f.value[1])
		}

		// No error, unmarshal the result part (first element) into valuePtr
		if f.value[0] != nil {
			// Convert the result part back to JSON and then unmarshal into valuePtr
			resultJSON, err := json.Marshal(f.value[0])
			if err != nil {
				return err
			}
			return json.Unmarshal(resultJSON, resultPtr)
		}

	}
	return nil
}
