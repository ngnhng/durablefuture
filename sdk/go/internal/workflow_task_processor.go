package internal

import (
	"context"
	"iter"
)

var _ WorkflowTaskProcessor = (*Conn)(nil)

// ReceiveTask implements WorkflowTaskProcessor.
func (c *Conn) ReceiveTask(ctx context.Context) iter.Seq[*TaskToken] {
	panic("unimplemented")
}
