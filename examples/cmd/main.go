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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"

	"github.com/ngnhng/durablefuture/examples/scenarios"
	_ "github.com/ngnhng/durablefuture/examples/scenarios/order"
	_ "github.com/ngnhng/durablefuture/examples/scenarios/recovery"
	clientpkg "github.com/ngnhng/durablefuture/pkg/client-sdk/client"
	"github.com/ngnhng/durablefuture/pkg/client-sdk/worker"
)

func main() {
	workerType := flag.String("worker", "client", "Type of worker to run: 'workflow', 'activity', 'both', or 'client'")
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS connection URL")
	exampleName := flag.String("example", "order", "Example scenario to execute")
	listExamples := flag.Bool("list-examples", false, "List available example scenarios and exit")
	flag.Parse()

	if *listExamples {
		log.Printf("available examples: %s", strings.Join(scenarios.Names(), ", "))
		return
	}

	example, ok := scenarios.Get(*exampleName)
	if !ok {
		log.Fatalf("unknown example '%s'. Available: %s", *exampleName, strings.Join(scenarios.Names(), ", "))
	}

	ctx := context.Background()

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("connecting to NATS failed: %v", err)
	}
	defer nc.Close()

	workflowClient, err := clientpkg.NewClient(&clientpkg.Options{
		Conn: nc,
	})
	if err != nil {
		log.Fatalf("creating workflow client failed: %v", err)
	}

	switch strings.ToLower(*workerType) {
	case "workflow":
		if err := runWorkflowWorker(ctx, workflowClient, example); err != nil {
			log.Fatalf("workflow worker exited: %v", err)
		}
	case "activity":
		if err := runActivityWorker(ctx, workflowClient, example); err != nil {
			log.Fatalf("activity worker exited: %v", err)
		}
	case "both":
		g, gCtx := errgroup.WithContext(ctx)
		g.Go(func() error { return runWorkflowWorker(gCtx, workflowClient, example) })
		g.Go(func() error { return runActivityWorker(gCtx, workflowClient, example) })
		if err := g.Wait(); err != nil {
			log.Fatalf("workers exited: %v", err)
		}
	case "client":
		if err := example.RunClient(ctx, workflowClient); err != nil {
			log.Fatalf("client execution failed: %v", err)
		}
	default:
		log.Fatalf("invalid worker type: %s. Use 'workflow', 'activity', 'both', or 'client'", *workerType)
	}
}

func runWorkflowWorker(ctx context.Context, c clientpkg.Client, example scenarios.Example) error {
	workerClient, err := worker.NewWorker(c, nil)
	if err != nil {
		return fmt.Errorf("error creating workflow worker: %w", err)
	}

	if err := example.RegisterWorkflows(workerClient); err != nil {
		return fmt.Errorf("error registering workflows: %w", err)
	}

	if err := workerClient.Run(ctx); err != nil {
		return fmt.Errorf("error running workflow worker: %w", err)
	}
	return nil
}

func runActivityWorker(ctx context.Context, c clientpkg.Client, example scenarios.Example) error {
	workerClient, err := worker.NewWorker(c, nil)
	if err != nil {
		return fmt.Errorf("error creating activity worker: %w", err)
	}

	if err := example.RegisterActivities(workerClient); err != nil {
		return fmt.Errorf("error registering activities: %w", err)
	}

	if err := workerClient.Run(ctx); err != nil {
		return fmt.Errorf("error running activity worker: %w", err)
	}
	return nil
}
