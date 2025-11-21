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

package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/ngnhng/durablefuture/api"
	natz "github.com/ngnhng/durablefuture/sdk/internal/protocol/nats"
)

// ExecuteWorkflow implements internal.WorkflowExecutor.
func StartWorkflow(ctx context.Context, c *natz.Conn, attrs *api.StartWorkflowAttributes) (*api.StartWorkflowReply, error) {
	attrsBytes, err := c.SerializeBinary(attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize start workflow attributes: %w", err)
	}

	command := api.Command{
		CommandType: api.StartWorkflowCommand,
		Attributes:  attrsBytes,
	}

	commandData, err := c.SerializeBinary(command)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize start workflow command: %w", err)
	}

	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	reply, err := c.NATS().RequestWithContext(reqCtx, api.CommandRequestStart, commandData)
	if err != nil {
		if errors.Is(err, nats.ErrNoResponders) {
			return nil, fmt.Errorf("no managers available to start workflow: %w", err)
		}
		return nil, fmt.Errorf("failed to send start workflow request: %w", err)
	}

	var parsedReply api.StartWorkflowReply
	if err := c.DeserializeBinary(reply.Data, &parsedReply); err != nil {
		return nil, fmt.Errorf("failed to parse reply of start workflow request: %w", err)
	}

	return &parsedReply, nil
}
