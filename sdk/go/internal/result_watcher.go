package internal

import (
	"context"
	"fmt"
	"log"

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
	log.Printf("Watching for result of workflow: %s", workflowID)

	// Wait for the first update or context cancellation.
	for update := range watcher.Updates() {
		if update == nil {
			// Channel was closed, likely due to context cancellation.
			continue
		}

		if update.Operation() == jetstream.KeyValuePut {
			log.Printf("Received result for workflow %s", workflowID)
			return update.Value(), nil
		}
	}

	// If the loop exits, the watcher was stopped, likely due to the context.
	return nil, fmt.Errorf("watcher stopped without receiving a result: %w", ctx.Err())
}
