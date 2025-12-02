package internal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	"github.com/ngnhng/durablefuture/sdk/internal/common"
	"github.com/ngnhng/durablefuture/sdk/internal/protocol/commands"
	natz "github.com/ngnhng/durablefuture/sdk/internal/protocol/nats"
)

// Client is the public interface for interacting with workflows.
// Users of the SDK will use this interface to start workflows and get results.
type Client interface {
	// ExecuteWorkflow starts a workflow execution and returns a Future for the result.
	// The workflowFn must be a function that has been registered with a worker.
	// Input parameters are passed to the workflow function and must be serializable.
	ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (Future, error)
}

type client interface {
	Client
	getConn() *natz.Conn
	getSerde() serde.BinarySerde
	getLogger() *slog.Logger
}

// ClientOptions contains configuration for creating a new Client.
type ClientOptions struct {
	// Namespace provides logical isolation between environments or tenants.
	// All workflows and activities are scoped to a namespace.
	Namespace string

	// Conn is the established NATS connection. Required.
	Conn *nats.Conn

	// Logger is used for SDK logging. If nil, a default logger is used.
	Logger *slog.Logger
}

var (
	_ Client = (*clientImpl)(nil)
	_ client = (*clientImpl)(nil)
)

type clientImpl struct {
	converter serde.BinarySerde
	logger    *slog.Logger
	options   *ClientOptions
	nc        *natz.Conn
}

func NewClient(options *ClientOptions) (Client, error) {
	if options == nil || options.Conn == nil {
		return nil, fmt.Errorf("client options must include an established NATS connection")
	}

	serder := &serde.MsgpackSerde{}
	logger := common.DefaultLogger(options.Logger)
	conn, err := natz.WrapExisting(options.Conn, options.Namespace, serder)
	if err != nil {
		return nil, err
	}
	conn.SetLogger(logger)

	return &clientImpl{
		converter: serder,
		logger:    logger,
		options:   options,
		nc:        conn,
	}, nil
}

func (c *clientImpl) startWorkflow(ctx context.Context, attrs *api.StartWorkflowAttributes) (*api.StartWorkflowReply, error) {
	return commands.StartWorkflow(ctx, c.nc, attrs)
}

// ExecuteWorkflow implements Client.
// It starts a workflow execution by sending a command request to the manager and waits for the reply containing the workflow ID.
func (c *clientImpl) ExecuteWorkflow(ctx context.Context, workflowFn any, input ...any) (Future, error) {
	workflowName, err := common.ExtractFullFunctionName(workflowFn)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow function name: %w", err)
	}

	attrs := &api.StartWorkflowAttributes{
		WorkflowFnName: workflowName,
		Input:          input,
	}

	reply, err := c.startWorkflow(ctx, attrs)

	if reply.Error != "" {
		return nil, fmt.Errorf("starting workflow failed on server: %s", reply.Error)
	}

	if reply.WorkflowID == "" {
		return nil, fmt.Errorf("empty workflowID: %w", err)
	}

	workflowId, _ := uuid.FromString(reply.WorkflowID)

	return newExecution(c, c.getConn(), c.converter, workflowId), nil
}

func (c *clientImpl) getConn() *natz.Conn         { return c.nc }
func (c *clientImpl) getSerde() serde.BinarySerde { return c.converter }
func (c *clientImpl) getLogger() *slog.Logger     { return c.logger }
