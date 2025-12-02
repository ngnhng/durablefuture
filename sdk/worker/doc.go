// Package worker provides the worker runtime for executing workflows and activities.
//
// Workers are processes that execute workflow and activity code. They connect
// to the DurableFuture server via NATS and poll for tasks.
//
// # Creating a Worker
//
// To create a worker, you need a client and worker options:
//
//	nc, err := nats.Connect("nats://localhost:4222")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	client, err := client.NewClient(&client.Options{
//		Namespace: "production",
//		Conn:      nc,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	worker, err := worker.NewWorker(client, &worker.Options{
//		Namespace: "production",
//		Logger:    slog.Default(),
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Registering Workflows and Activities
//
// Before running the worker, register your workflows and activities:
//
//	// Register workflows
//	err = worker.RegisterWorkflow(MyWorkflow)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Register activities
//	err = worker.RegisterActivity(MyActivity)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Running the Worker
//
// Start the worker to begin processing tasks:
//
//	ctx := context.Background()
//	if err := worker.Run(ctx); err != nil {
//		log.Fatal(err)
//	}
//
// The worker will run until the context is canceled or an error occurs.
//
// # Workflow Workers vs Activity Workers
//
// A single worker can execute both workflows and activities. In production,
// you may want to run separate workers for workflows and activities to
// scale them independently.
//
// # Graceful Shutdown
//
// To gracefully shut down a worker, cancel its context:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	// Handle shutdown signal
//	go func() {
//		<-shutdownSignal
//		cancel()
//	}()
//
//	if err := worker.Run(ctx); err != nil && err != context.Canceled {
//		log.Fatal(err)
//	}
//
// # Worker Scaling
//
// You can run multiple worker processes to increase throughput. Each worker
// competes for tasks from the same NATS streams.
package worker
