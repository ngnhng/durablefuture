package internal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/sdk/internal/common"
)

var _ Future = (*blocking)(nil)

type Future interface {
	Get(ctx context.Context, valuePtr any) error
}

type blocking struct {
	isResolved bool
	value      []any
	err        error
	converter  serde.BinarySerde
	logger     *slog.Logger
}

func (f *blocking) Get(ctx context.Context, resultPtr any) error {
	if !f.isResolved {
		panic(errorBlockingFuture{})
	}
	if f.err != nil {
		return f.err
	}
	if resultPtr != nil && f.value != nil {

		f.loggerOrDefault().Debug("activity future resolved", "values", common.DebugAnyValues(f.value))

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

		// No error, convert the result part (first element) into valuePtr
		if f.value[0] != nil {
			// Use serialization-agnostic type conversion
			// This works regardless of the underlying serializer (JSON, msgpack, protobuf)
			if f.converter == nil {
				return fmt.Errorf("no converter available for type conversion")
			}

			// Serialize the value using the configured serializer
			resultBytes, err := f.converter.SerializeBinary(f.value[0])
			if err != nil {
				return fmt.Errorf("failed to serialize result value: %w", err)
			}

			// Deserialize into the target type
			if err := f.converter.DeserializeBinary(resultBytes, resultPtr); err != nil {
				return fmt.Errorf("failed to deserialize result into target type: %w", err)
			}

			return nil
		}

	}
	return nil
}

func (f *blocking) loggerOrDefault() *slog.Logger {
	if f == nil {
		return slog.Default()
	}
	return common.DefaultLogger(f.logger)
}
