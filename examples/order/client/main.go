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
	"time"

	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"

	"github.com/ngnhng/durablefuture/pkg/client-sdk/client"
	"github.com/ngnhng/durablefuture/pkg/client-sdk/worker"
	"github.com/ngnhng/durablefuture/pkg/client-sdk/workflow"
)

// OrderInfo represents the input to the order workflow.
type OrderInfo struct {
	CustomerID string  `json:"customer_id"`
	ProductID  string  `json:"product_id"`
	Amount     float64 `json:"amount"`
	Quantity   int     `json:"quantity"`
}

// ChargeResult represents the result of charging a credit card.
type ChargeResult struct {
	ChargeID      string  `json:"charge_id"`
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transaction_id"`
}

// ShipResult represents the result of shipping a package.
type ShipResult struct {
	TrackingID        string    `json:"tracking_id"`
	Carrier           string    `json:"carrier"`
	EstimatedDelivery time.Time `json:"estimated_delivery"`
}

// OrderWorkflow implements a simple order processing workflow.
func OrderWorkflow(ctx workflow.Context, customerID string, productID string, amount float64, quantity int) (any, error) {
	log.Println("order workflow started")

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         nil,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var chargeResult ChargeResult
	if err := workflow.
		ExecuteActivity(ctx, ChargeCreditCardActivity, customerID, amount).
		Get(ctx, &chargeResult); err != nil {
		return nil, fmt.Errorf("credit card charge failed: %w", err)
	}

	var shipResult ShipResult
	if err := workflow.
		ExecuteActivity(ctx, ShipPackageActivity, chargeResult).
		Get(ctx, &shipResult); err != nil {
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

// ChargeCreditCardActivity simulates charging a credit card.
func ChargeCreditCardActivity(ctx context.Context, customerID string, amount float64) (any, error) {
	log.Printf("charge credit card activity")

	time.Sleep(10 * time.Second)

	if customerID == "fail_customer" {
		return nil, fmt.Errorf("credit card declined")
	}

	result := ChargeResult{
		ChargeID:      fmt.Sprintf("charge_%d", time.Now().UnixNano()),
		Amount:        amount,
		TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
	}

	return result, nil
}

// ShipPackageActivity simulates shipping a package.
func ShipPackageActivity(ctx context.Context, chargeResult ChargeResult) (any, error) {
	log.Printf("ship activity")

	time.Sleep(3 * time.Second)

	result := ShipResult{
		TrackingID:        fmt.Sprintf("track_%d", time.Now().UnixNano()),
		Carrier:           "FastShip Express",
		EstimatedDelivery: time.Now().Add(3 * 24 * time.Hour),
	}

	return result, nil
}

// DelayedActivity is used in the timer workflow example.
func DelayedActivity(ctx context.Context, input any) (any, error) {
	time.Sleep(1 * time.Second)

	result := map[string]any{
		"message":   "Delayed activity completed",
		"timestamp": time.Now(),
		"input":     input,
	}

	return result, nil
}

// AddActivity is a simple activity for testing.
func AddActivity(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type for AddActivity")
	}

	a, ok1 := inputMap["a"].(float64)
	b, ok2 := inputMap["b"].(float64)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("invalid numeric inputs")
	}

	return a + b, nil
}

func main() {
	workerType := flag.String("worker", "both", "Type of worker to run: 'workflow', 'activity', 'both', or 'client'")
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS connection URL")
	flag.Parse()

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

	switch *workerType {
	case "workflow":
		if err := runWorkflowWorker(ctx, workflowClient); err != nil {
			log.Fatalf("workflow worker exited: %v", err)
		}
	case "activity":
		if err := runActivityWorker(ctx, workflowClient); err != nil {
			log.Fatalf("activity worker exited: %v", err)
		}
	case "both":
		g, gCtx := errgroup.WithContext(ctx)
		g.Go(func() error { return runWorkflowWorker(gCtx, workflowClient) })
		g.Go(func() error { return runActivityWorker(gCtx, workflowClient) })
		if err := g.Wait(); err != nil {
			log.Fatalf("workers exited: %v", err)
		}
	case "client":
		if err := runClient(ctx, workflowClient); err != nil {
			log.Fatalf("client execution failed: %v", err)
		}
	default:
		log.Fatalf("invalid worker type: %s. Use 'workflow', 'activity', 'both', or 'client'", *workerType)
	}
}

func runWorkflowWorker(ctx context.Context, c client.Client) error {
	workerClient, err := worker.NewWorker(c, nil)
	if err != nil {
		return fmt.Errorf("error creating workflow worker: %w", err)
	}

	log.Println("registering workflow")
	if err := workerClient.RegisterWorkflow(OrderWorkflow); err != nil {
		return fmt.Errorf("error registering workflow: %w", err)
	}

	if err := workerClient.Run(ctx); err != nil {
		return fmt.Errorf("error running workflow worker: %w", err)
	}
	return nil
}

func runActivityWorker(ctx context.Context, c client.Client) error {
	workerClient, err := worker.NewWorker(c, nil)
	if err != nil {
		return fmt.Errorf("error creating activity worker: %w", err)
	}

	log.Println("registering activities")
	register := []struct {
		name string
		fn   any
	}{
		{"AddActivity", AddActivity},
		{"DelayedActivity", DelayedActivity},
		{"ChargeCreditCardActivity", ChargeCreditCardActivity},
		{"ShipPackageActivity", ShipPackageActivity},
	}

	for _, entry := range register {
		if err := workerClient.RegisterActivity(entry.fn); err != nil {
			return fmt.Errorf("error registering %s: %w", entry.name, err)
		}
	}

	if err := workerClient.Run(ctx); err != nil {
		return fmt.Errorf("error running activity worker: %w", err)
	}
	return nil
}

func runClient(ctx context.Context, workflowClient client.Client) error {
	future, err := workflowClient.ExecuteWorkflow(ctx, OrderWorkflow,
		"Bob",
		"widget-1000",
		1000.0,
		2,
	)
	if err != nil {
		return fmt.Errorf("starting workflow failed: %w", err)
	}

	var result any
	if err := future.Get(ctx, &result); err != nil {
		return fmt.Errorf("waiting for workflow result failed: %w", err)
	}

	log.Printf("workflow result: %v", result)
	return nil
}
