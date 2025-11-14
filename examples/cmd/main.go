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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"

	"github.com/ngnhng/durablefuture/examples/scenarios"
	_ "github.com/ngnhng/durablefuture/examples/scenarios/order"
	_ "github.com/ngnhng/durablefuture/examples/scenarios/order-retries"
	_ "github.com/ngnhng/durablefuture/examples/scenarios/recovery"
	"github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/worker"
)

func main() {
	workerType := flag.String("worker", "client", "Type of worker to run: 'workflow', 'activity', 'both', or 'client'")
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS connection URL")
	exampleName := flag.String("example", "order", "Example scenario to execute")
	listExamples := flag.Bool("list-examples", false, "List available example scenarios and exit")
	httpPort := flag.String("http-port", "", "HTTP port for health/crash endpoints (optional)")
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

	workflowClient, err := client.NewClient(&client.Options{
		Conn: nc,
	})
	if err != nil {
		log.Fatalf("creating workflow client failed: %v", err)
	}

	// Start HTTP server if port is specified (for demo crash/health endpoints)
	g, gCtx := errgroup.WithContext(ctx)
	if *httpPort != "" {
		g.Go(func() error {
			return startHTTPServer(gCtx, *httpPort, *workerType)
		})
	}

	switch strings.ToLower(*workerType) {
	case "workflow":
		g.Go(func() error {
			return runWorkflowWorker(gCtx, workflowClient, example)
		})
	case "activity":
		g.Go(func() error {
			return runActivityWorker(gCtx, workflowClient, example)
		})
	case "both":
		g.Go(func() error { return runWorkflowWorker(gCtx, workflowClient, example) })
		g.Go(func() error { return runActivityWorker(gCtx, workflowClient, example) })
	case "client":
		if err := example.RunClient(ctx, workflowClient); err != nil {
			log.Fatalf("client execution failed: %v", err)
		}
		return
	default:
		log.Fatalf("invalid worker type: %s. Use 'workflow', 'activity', 'both', or 'client'", *workerType)
	}

	if err := g.Wait(); err != nil {
		log.Fatalf("worker exited: %v", err)
	}
}

func runWorkflowWorker(ctx context.Context, c client.Client, example scenarios.Example) error {
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

func runActivityWorker(ctx context.Context, c client.Client, example scenarios.Example) error {
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

// startHTTPServer starts a simple HTTP server for health checks and crash endpoints
func startHTTPServer(ctx context.Context, port string, workerType string) error {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "healthy",
			"worker_type": workerType,
		})
	})

	// Crash endpoint
	mux.HandleFunc("/crash", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[%s worker] Crash requested via HTTP endpoint", workerType)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "crashing",
			"message": fmt.Sprintf("%s worker will exit", workerType),
		})

		// Give response time to send
		go func() {
			time.Sleep(100 * time.Millisecond)
			log.Printf("[%s worker] Exiting...", workerType)
			os.Exit(1)
		}()
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: corsMiddleware(mux),
	}

	log.Printf("[%s worker] HTTP server listening on :%s", workerType, port)

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return server.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
