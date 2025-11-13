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

package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/sdk/internal/utils"
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
	converter  serde.BinarySerde // for serialization-agnostic type conversion
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
