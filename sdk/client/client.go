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

package client

import "github.com/ngnhng/durablefuture/sdk/internal"

// Client is the interface for interacting with DurableFuture workflows.
//
// Use Client to start workflow executions and retrieve results. A client connects
// to the DurableFuture server via NATS and provides a type-safe API for workflow operations.
//
// Example:
//
//	client, err := client.NewClient(&client.Options{
//		Namespace: "production",
//		Conn:      natsConn,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	future, err := client.ExecuteWorkflow(ctx, MyWorkflow, arg1, arg2)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	var result MyResult
//	if err := future.Get(ctx, &result); err != nil {
//		log.Fatal(err)
//	}
type Client = internal.Client

// Options contains configuration for creating a new Client.
type Options = internal.ClientOptions

// NewClient creates a new Client with the provided Options.
//
// The Options must include an established NATS connection. The client will use
// this connection to communicate with the DurableFuture server.
//
// Returns an error if:
//   - Options is nil
//   - Options.Conn is nil
//   - Unable to establish JetStream context
func NewClient(options *Options) (Client, error) {
	return internal.NewClient(options)
}
