package internal

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/server/utils"
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
		getConn() *Conn
		getSerde() serde.BinarySerde
	}

	ClientOptions struct {
		Namespace string
		Conn      *nats.Conn
	}
)

type clientImpl struct {
	WorkflowExecutor

	converter serde.BinarySerde
	options   *ClientOptions
	nc        *Conn
}

func NewClient(options *ClientOptions) (Client, error) {

	conn, err := from(options.Conn)
	if err != nil {
		return nil, err
	}

	return &clientImpl{
		WorkflowExecutor: conn,
		converter:        &serde.JsonSerde{},
		options:          options,
	}, nil
}

// ExecuteWorkflow implements Client.
// It starts a workflow execution by sending a command request to the manager and waits for the reply containing the workflow ID.
func (c *clientImpl) ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (Future, error) {

	workflowName, err := utils.ExtractFullFunctionName(workflowFn)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow function name: %w", err)
	}

	attrs := &api.StartWorkflowAttributes{
		WorkflowFnName: workflowName,
		Input:          input,
	}

	reply, err := c.StartWorkflow(ctx, attrs)

	// commandData, err := c.converter.SerializeBinary(startWorkflowCommand)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to serialize start workflow command: %w", err)
	// }

	// // Use configured request timeout; fall back to 10s if zero (defensive)
	// timeout := c.cfg.Timeouts.RequestTimeout
	// reqCtx, cancel := context.WithTimeout(ctx, timeout)
	// defer cancel()

	// log.Printf("request: %v", startWorkflowCommand)
	// reply, err := c.conn.NATS().RequestWithContext(reqCtx, api.CommandRequestStart, commandData)
	// if err != nil {
	// 	log.Printf("err: %v", err)
	// 	if errors.Is(err, nats.ErrNoResponders) {
	// 		return nil, fmt.Errorf("no managers available to start workflow: %w", err)
	// 	}
	// 	return nil, fmt.Errorf("failed to send start workflow request: %w", err)
	// }

	// err = c.converter.DeserializeBinary(reply.Data, &parsedReply)
	// if err != nil {
	// 	log.Printf("err: %v", err)
	// 	return nil, fmt.Errorf("failed to parse reply of start workflow request: %w", err)
	// }

	if reply.Error != "" {
		return nil, fmt.Errorf("starting workflow failed on server: %s", reply.Error)
	}

	if reply.WorkflowID == "" {
		return nil, fmt.Errorf("empty workflowID: %w", err)
	}

	workflowId, _ := uuid.FromString(reply.WorkflowID)

	return NewExecution(c, nil, c.converter, workflowId), nil
}

func (c *clientImpl) getConn() *Conn              { return c.nc }
func (c *clientImpl) getSerde() serde.BinarySerde { return c.converter }
