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
