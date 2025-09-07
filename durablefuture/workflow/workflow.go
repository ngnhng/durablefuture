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
	"encoding/json"
	"fmt"
	"log"

	"durablefuture/internal/converter"
	"durablefuture/internal/natz"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go/jetstream"
)

type Execution struct {
	WorkflowID string
	converter  converter.Converter
	*natz.Conn
}

func NewExecution(c *natz.Conn, conv converter.Converter, workflowID uuid.UUID) *Execution {
	if conv == nil {
		return &Execution{
			WorkflowID: workflowID.String(),
			Conn:       c,
			converter:  converter.NewJsonConverter(),
		}
	}
	return &Execution{
		WorkflowID: workflowID.String(),
		Conn:       c,
		converter:  conv,
	}
}

func (w *Execution) Get(ctx context.Context, valuePtr any) error {

	watcher, err := w.Conn.WatchKV(ctx, "workflow-result", w.WorkflowID)
	if err != nil {
		log.Printf("error waiting for workflow result: %v", err)
	}

	defer watcher.Stop()

	for update := range watcher.Updates() {
		if update == nil {
			continue
		}

		if update.Operation() == jetstream.KeyValueDelete {
		} else {
			// Predefined convention: The workflow result contains [result, error] array
			var resultArray []any
			if err := w.converter.From(update.Value(), &resultArray); err != nil {
				return err
			}

			if len(resultArray) != 2 {
				return fmt.Errorf("invalid workflow result format: expected [result, error], got %d elements", len(resultArray))
			}

			// Extract the error part (second element)
			if resultArray[1] != nil {
				if errStr, ok := resultArray[1].(string); ok && errStr != "" {
					return fmt.Errorf("%s", errStr) // TODO: errors.Is friendly
				}
				return fmt.Errorf("sdk error: %v", resultArray[1])
			}

			if valuePtr != nil && resultArray[0] != nil {
				resultJSON, err := json.Marshal(resultArray[0])
				if err != nil {
					return err
				}
				return json.Unmarshal(resultJSON, valuePtr)
			}

			return nil
		}
	}

	return nil

}

func (w *Execution) IsReady() bool {
	return false
}

var _ Future = (*Execution)(nil)
