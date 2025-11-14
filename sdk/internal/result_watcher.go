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

	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
)

// Watch implements internal.WorkflowResultWatcher.
func (c *Conn) Watch(ctx context.Context, workflowID string) ([]byte, error) {
	watcher, err := c.WatchKV(ctx, api.WorkflowResultBucket, workflowID)
	if err != nil {
		return nil, fmt.Errorf("could not start KV watcher for key '%s': %w", workflowID, err)
	}
	defer watcher.Stop()
	c.Logger().Info("watching for workflow result", "workflow_id", workflowID)

	// Wait for the first update or context cancellation.
	for update := range watcher.Updates() {
		if update == nil {
			// Channel was closed, likely due to context cancellation.
			continue
		}

		if update.Operation() == jetstream.KeyValuePut {
			c.Logger().Info("received workflow result", "workflow_id", workflowID)
			return update.Value(), nil
		}
	}

	// If the loop exits, the watcher was stopped, likely due to the context.
	return nil, fmt.Errorf("watcher stopped without receiving a result: %w", ctx.Err())
}
