package worker

import (
	"context"

	"github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/internal"
)

// Worker is the interface for the worker runtime that executes workflows and activities.
//
// A worker polls for tasks from NATS, executes workflow and activity code, and
// reports results back to the server. Workers must register workflows and activities
// before starting.
//
// Example:
//
//	worker, err := worker.NewWorker(client, &worker.Options{
//		Namespace: "production",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Register workflows and activities
//	worker.RegisterWorkflow(MyWorkflow)
//	worker.RegisterActivity(MyActivity)
//
//	// Run the worker
//	if err := worker.Run(ctx); err != nil {
//		log.Fatal(err)
//	}
type Worker interface {
	Registry
	// Run starts the worker and blocks until the context is canceled or an error occurs.
	// The worker will continuously poll for and process workflow and activity tasks.
	Run(ctx context.Context) error
}

// Registry combines workflow and activity registration interfaces.
type Registry interface {
	WorkflowRegistry
	ActivityRegistry
}

// WorkflowRegistry provides methods for registering workflow functions.
//
// Workflows must be registered before the worker starts. The workflow function
// signature should be: func(workflow.Context, ...args) (result, error)
type WorkflowRegistry = internal.WorkflowRegistry

// ActivityRegistry provides methods for registering activity functions.
//
// Activities must be registered before the worker starts. The activity function
// signature should be: func(context.Context, ...args) (result, error)
type ActivityRegistry = internal.ActivityRegistry

// Options contains configuration for creating a new Worker.
type Options = internal.WorkerOptions

// NewWorker creates a new Worker with the provided client and options.
//
// The worker uses the client's NATS connection to communicate with the server.
// Returns an error if unable to create the worker or establish required streams.
func NewWorker(c client.Client, options *Options) (Worker, error) {
	return internal.NewWorker(c, options)
}
