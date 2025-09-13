# DurableFuture

A lightweight workflow engine built on **NATS JetStream** for durable, distributed task orchestration.

You can focus on your business logic while the engine handles persistence, retries, and distributed execution.

> [!WARNING]
> Software is at pre-alpha stage and is just a concept demo, expect bugs.

### Architecture

- **Server** - Orchestrates workflow execution and state management
- **Workflow Worker** - Executes workflow logic and decision-making
- **Activity Worker** - Performs individual tasks (API calls, database operations, etc.)
- **Client** - Triggers workflow execution and retrieves results

## Why DurableFuture?

Traditional workflow engines often require complex deployments with proprietary databases and clustering solutions. DurableFuture simplifies this by:

- **Delegating persistence to NATS JetStream** - No custom database layer
- **Minimal operational overhead** - Just NATS + your application
- **Production-ready foundation** - NATS is proven in high-scale environments

## Usage

- Workflow: See full version at [/examples/order.go](durablefuture/examples/order.go)
- Retry Examples: See retry configurations at [/examples/retry_examples.go](durablefuture/examples/retry_examples.go)
- Usage: See full version at [/usage/main.go](usage/main.go)

### 1. Write a workflow

```go
// OrderWorkflow implements a simple order processing workflow
func OrderWorkflow(ctx workflow.Context, customerId string, productId string, amount float64, quantity int) (any, error) {

	var chargeResult ChargeResult
	if err := workflow.
		ExecuteActivity(ctx, ChargeCreditCardActivity, customerId, amount).
		Get(ctx, &chargeResult); err != nil {
		return nil, fmt.Errorf("credit card charge failed: %w", err)
	}

	var shipResult ShipResult
	if err := workflow.
		ExecuteActivity(ctx, ShipPackageActivity, chargeResult).
		Get(ctx, &shipResult); err != nil {
		// In a real scenario, you would refund the charge here
		return nil, fmt.Errorf("package shipping failed: %w", err)
	}

	result := map[string]any{
		"tracking_id":        shipResult.TrackingID,
		"carrier":            shipResult.Carrier,
		"estimated_delivery": shipResult.EstimatedDelivery,
		"charge_id":          chargeResult.ChargeID,
	}

	return result, nil
}
```

### 2. Create Workers and register said Workflow

- First create Workflow Worker:

```go
	ctx := context.Background()
	// create a Workflow Worker
	workerClient, err := worker.NewWorker()
	if err != nil {
		log.Printf("err: %v", err)
		return
	}
	// register the Workflow to the Worker
	err = workerClient.RegisterWorkflow(examples.OrderWorkflow)
	if err != nil {
		log.Printf("err: %v", err)
		return
	}
	// start the Workflow Worker
	if err := workerClient.Run(ctx); err != nil {
		log.Printf("err: %v", err)
		return
	}
```

- Then create Activity Worker(s):

```go
	ctx := context.Background()
	workerClient, err := worker.NewWorker()
	if err != nil {
		return
	}
	err = workerClient.RegisterActivity(examples.AddActivity)
	if err != nil {
		return
	}
	err = workerClient.RegisterActivity(examples.DelayedActivity)
	if err != nil {
		return
	}
	err = workerClient.RegisterActivity(examples.ChargeCreditCardActivity)
	if err != nil {
		return
	}
	err = workerClient.RegisterActivity(examples.ShipPackageActivity)
	if err != nil {
		return
	}
	if err := workerClient.Run(ctx); err != nil {
		return
	}
```

## Configuration Features

### Activity Retry Policies

DurableFuture supports sophisticated retry policies for activities:

```go
// Exponential backoff retry policy
retryPolicy := &types.RetryPolicy{
    MaximumAttempts:    5,
    InitialInterval:    1 * time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    30 * time.Second,
    BackoffPolicy:      types.BackoffExponential,
    NonRetryableErrorTypes: []string{"fatal error", "validation error"},
}

activityOptions := &types.ActivityOptions{
    TaskQueue:              "default",
    ScheduleToCloseTimeout: 5 * time.Minute,
    StartToCloseTimeout:    30 * time.Second,
    RetryPolicy:            retryPolicy,
}

// Execute activity with custom retry policy
var result string
err := workflow.ExecuteActivityWithOptions(ctx, MyActivity, activityOptions, input).
    Get(ctx, &result)
```

### Backoff Strategies

Three backoff strategies are supported:

1. **Exponential Backoff** (`BackoffExponential`): Delay increases exponentially (1s, 2s, 4s, 8s...)
2. **Linear Backoff** (`BackoffLinear`): Delay increases linearly (1s, 2s, 3s, 4s...)
3. **Fixed Backoff** (`BackoffFixed`): Constant delay between retries (1s, 1s, 1s, 1s...)

### Timeout Configurations

Activities support multiple timeout configurations:

- **ScheduleToCloseTimeout**: Total time allowed for the activity
- **StartToCloseTimeout**: Time allowed for execution once started
- **ScheduleToStartTimeout**: Time allowed to wait for a worker
- **HeartbeatTimeout**: Heartbeat interval for long-running activities

### Namespace Support

DurableFuture supports namespaces for multi-tenant isolation:

```go
workflowOptions := &types.WorkflowOptions{
    TaskQueue: "default",
    Namespace: "production",
    WorkflowExecutionTimeout: 10 * time.Minute,
}
```

### Non-Retryable Errors

Specify error types that should not trigger retries:

```go
retryPolicy := &types.RetryPolicy{
    MaximumAttempts: 3,
    NonRetryableErrorTypes: []string{
        "fatal error",
        "validation error", 
        "authentication failed",
    },
}
```

- Finally, the client code:

```go
future, err := workflowClient.ExecuteWorkflow(ctx, examples.OrderWorkflow,
	"Bob",
	"widget-1000",
	1000.0,
	2,
)
if err != nil {
	log.Fatalf("Starting workflow failed: %v", err)
}
var result any
err = future.Get(ctx, &result)
if err != nil {
	log.Fatalf("error: %v", err)
}
log.Printf("result: %v", result)
```

## How it works

DurableFuture works by leveraging the event sourcing pattern, recording the outcome of Activities within the Workflow. For example, results of operations such as making an API call or a database transaction will be persisted as Events in a NATS Jetstream. So in the event the current Workflow crashed or interrupted, it will be re-run on one of the available Workflow Workers. However, instead of executing the Activities that have already been done, it will return the result from the first successful execution that is being stored on the Event Stream.

Consider the previous example:

If the workflow runs normally without being interrupted, then the event log at the end might look something like this:

| seq | event type         | result                                       |
| --- | ------------------ | -------------------------------------------- |
| 0   | workflow started   | Order                                        |
| 1   | activity scheduled | ChargeCreditCard                             |
| 2   | activity started   | ChargeCreditCard                             |
| 3   | activity completed | {charge_id: "ch_123"}                        |
| 4   | activity scheduled | Shipping                                     |
| 5   | activity started   | Shipping                                     |
| 6   | activity completed | {tracking_id: "tr_456"}                      |
| 7   | workflow completed | {charge_id: "ch_123", tracking_id: "tr_456"} |

Suppose, now, that instead of running until the end, some failure occurs after the `ChargeCreditCard` activity has completed, but before the `Shipping` activity has completed. The event log might look like this:

| seq | event type         | result                                       |
| --- | ------------------ | -------------------------------------------- |
| 0   | workflow started   | Order                                        |
| 1   | activity scheduled | ChargeCreditCard                             |
| 2   | activity started   | ChargeCreditCard                             |
| 3   | activity completed | {charge_id: "ch_123"}                        |
| 4   | activity scheduled | Shipping                                     |
| 5   | activity started   | Shipping                                     |
| 6   | activity failed    | (crashed before completion)                  |
| 7   | activity scheduled | Shipping (retries)                           |
| 8   | activity started   | Shipping (retries)                           |
| 9   | activity completed | {tracking_id: "tr_456"}                      |
| 10  | workflow completed | {charge_id: "ch_123", tracking_id: "tr_456"} |

So when a Worker picks up the Workflow, it is restarted and it will replay the events in the log, only executing the Activities that have not yet been completed.
