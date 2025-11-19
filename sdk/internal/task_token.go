package internal

import (
	"context"

	"github.com/ngnhng/durablefuture/api"
)

type taskToken struct {
	Task api.Task
	Ack  func(context.Context) error
	Nak  func(context.Context) error
	Term func(context.Context) error
}
