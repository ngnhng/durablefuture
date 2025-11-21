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
	"log/slog"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
)

var _ Client = (*clientImpl)(nil)

type (
	WorkflowExecutor interface {
		StartWorkflow(ctx context.Context, attrs *api.StartWorkflowAttributes) (*api.StartWorkflowReply, error)
	}

	Client interface {
		// ExecuteWorkflow starts a workflow execution.
		ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (Future, error)
		// Accessors to underlying components, not exposed for public consumption
		getConn() *sdkNATSConnection
		getSerde() serde.BinarySerde
		getLogger() *slog.Logger
	}

	ClientOptions struct {
		Namespace string
		Conn      *nats.Conn
		Logger    *slog.Logger
	}
)

type clientImpl struct {
	WorkflowExecutor

	converter serde.BinarySerde
	logger    *slog.Logger
	options   *ClientOptions
	nc        *sdkNATSConnection
}

func NewClient(options *ClientOptions) (Client, error) {
	if options == nil || options.Conn == nil {
		return nil, fmt.Errorf("client options must include an established NATS connection")
	}

	serder := &serde.MsgpackSerde{}
	logger := defaultLogger(options.Logger)
	conn, err := wrapExisting(options.Conn, options.Namespace, serder)
	if err != nil {
		return nil, err
	}
	conn.SetLogger(logger)

	return &clientImpl{
		WorkflowExecutor: conn,
		converter:        serder,
		logger:           logger,
		options:          options,
		nc:               conn,
	}, nil
}

// ExecuteWorkflow implements Client.
// It starts a workflow execution by sending a command request to the manager and waits for the reply containing the workflow ID.
func (c *clientImpl) ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (Future, error) {

	workflowName, err := extractFullFunctionName(workflowFn)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow function name: %w", err)
	}

	attrs := &api.StartWorkflowAttributes{
		WorkflowFnName: workflowName,
		Input:          input,
	}

	reply, err := c.StartWorkflow(ctx, attrs)

	if reply.Error != "" {
		return nil, fmt.Errorf("starting workflow failed on server: %s", reply.Error)
	}

	if reply.WorkflowID == "" {
		return nil, fmt.Errorf("empty workflowID: %w", err)
	}

	workflowId, _ := uuid.FromString(reply.WorkflowID)

	return NewExecution(c, c.getConn(), c.converter, workflowId), nil
}

func (c *clientImpl) getConn() *sdkNATSConnection { return c.nc }
func (c *clientImpl) getSerde() serde.BinarySerde { return c.converter }
func (c *clientImpl) getLogger() *slog.Logger     { return c.logger }
