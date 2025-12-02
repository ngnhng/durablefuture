// Package client provides the client for interacting with DurableFuture workflows.
//
// The client package allows you to start workflow executions, wait for results,
// and interact with running workflows.
//
// # Creating a Client
//
// To create a client, you need an established NATS connection:
//
//	nc, err := nats.Connect("nats://localhost:4222")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	client, err := client.NewClient(&client.Options{
//		Namespace: "production",
//		Conn:      nc,
//		Logger:    slog.Default(),
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Executing Workflows
//
// Use ExecuteWorkflow to start a workflow execution:
//
//	future, err := client.ExecuteWorkflow(ctx, MyWorkflow, "input1", "input2")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	var result string
//	if err := future.Get(ctx, &result); err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("Result:", result)
//
// # Workflow Results
//
// The ExecuteWorkflow method returns a Future that you can use to wait for
// the workflow to complete and retrieve its result. The Get method blocks
// until the workflow completes or the context is canceled.
//
// # Namespaces
//
// Namespaces provide logical isolation between different environments or tenants.
// Workflows, activities, and results are scoped to a namespace.
package client
