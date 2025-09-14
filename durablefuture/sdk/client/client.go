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

package client

import (
	"context"
	"errors"
	"fmt"
	"log"

	"durablefuture/internal/config"
	"durablefuture/internal/constants"
	"durablefuture/internal/converter"
	"durablefuture/internal/natz"
	"durablefuture/internal/types"
	"durablefuture/internal/utils"
	"durablefuture/workflow"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go"
)

type Client interface {
	// ExecuteWorkflow starts a workflow execution.
	ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (workflow.Future, error)
}

type Config struct {
}

func NewClient() (Client, error) {
	return newClient()
}

type impl struct {
	conn      *natz.Conn
	converter converter.Converter
}

func newClient() (*impl, error) {
	conn, err := natz.Connect(natz.DefaultConfig())
	if err != nil {
		return nil, err
	}

	conv := converter.NewJsonConverter()

	return &impl{
		conn:      conn,
		converter: conv,
	}, nil
}

// ExecuteWorkflow implements Client.
// It starts a workflow execution by sending a command request to the manager and waits for the reply containing the workflow ID.
func (c *impl) ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (workflow.Future, error) {
	appConfig := config.LoadConfig()

	workflowName, err := utils.ExtractFullFunctionName(workflowFn)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow function name: %w", err)
	}

	attrs := &types.StartWorkflowAttributes{
		WorkflowFnName: workflowName,
		Input:          input,
	}

	attrsBytes, err := c.converter.To(attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize start workflow attributes: %w", err)
	}

	startWorkflowCommand := types.CommandEvent{
		CommandType: types.StartWorkflowCommand,
		Attributes:  attrsBytes,
	}

	commandData, err := c.converter.To(startWorkflowCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize start workflow command: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, appConfig.Timeouts.RequestTimeout)
	defer cancel()

	log.Printf("request: %v", startWorkflowCommand)
	reply, err := c.conn.NATS().RequestWithContext(reqCtx, constants.CommandRequestStart, commandData)
	if err != nil {
		log.Printf("err: %v", err)
		if errors.Is(err, nats.ErrNoResponders) {
			return nil, fmt.Errorf("no managers available to start workflow: %w", err)
		}
		return nil, fmt.Errorf("failed to send start workflow request: %w", err)
	}

	var parsedReply types.StartWorkflowReply
	err = c.converter.From(reply.Data, &parsedReply)
	if err != nil {
		log.Printf("err: %v", err)
		return nil, fmt.Errorf("failed to parse reply of start workflow request: %w", err)
	}

	log.Printf("parsed reply: %v", parsedReply)
	if parsedReply.Error != "" {
		return nil, fmt.Errorf("starting workflow failed on server: %s", parsedReply.Error)
	}

	if parsedReply.WorkflowID == "" {
		return nil, fmt.Errorf("empty workflowID: %w", err)
	}
	workflowId, _ := uuid.FromString(parsedReply.WorkflowID)
	return workflow.NewExecution(c.conn, c.converter, workflowId), nil
}
