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
	"github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/worker"
	"github.com/ngnhng/durablefuture/sdk/workflow"
)

// OrderInfo represents the input to the order workflow
type OrderInfo struct {
	CustomerID string  `json:"customer_id"`
	ProductID  string  `json:"product_id"`
	Amount     float64 `json:"amount"`
	Quantity   int     `json:"quantity"`
}

// ChargeResult represents the result of charging a credit card
type ChargeResult struct {
	ChargeID      string  `json:"charge_id"`
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transaction_id"`
}

// ShipResult represents the result of shipping a package
type ShipResult struct {
	TrackingID        string    `json:"tracking_id"`
	Carrier           string    `json:"carrier"`
	EstimatedDelivery time.Time `json:"estimated_delivery"`
}

// OrderWorkflow implements a simple order processing workflow
func OrderWorkflow(ctx workflow.Context, customerId string, productId string, amount float64, quantity int) (any, error) {
	log.Println("order workflow started")

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         nil,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var chargeResult ChargeResult
	if err := workflow.
		ExecuteActivity(ctx, nil, ChargeCreditCardActivity, customerId, amount).
		Get(ctx, &chargeResult); err != nil {
		return nil, fmt.Errorf("credit card charge failed: %w", err)
	}

	var shipResult ShipResult
	if err := workflow.
		ExecuteActivity(ctx, nil, ShipPackageActivity, chargeResult).
		Get(ctx, &shipResult); err != nil {
		// In a real scenario, you would refund the charge here
		return nil, fmt.Errorf("package shipping failed: %w", err)
	}

	// Return the tracking ID as the workflow result
	result := map[string]any{
		"tracking_id":        shipResult.TrackingID,
		"carrier":            shipResult.Carrier,
		"estimated_delivery": shipResult.EstimatedDelivery,
		"charge_id":          chargeResult.ChargeID,
	}

	return result, nil
}

// ChargeCreditCardActivity simulates charging a credit card
func ChargeCreditCardActivity(ctx context.Context, customerId string, amount float64) (any, error) {
	log.Printf("charge credit card activity")

	// Simulate some processing time
	time.Sleep(10 * time.Second)

	// Simulate potential failure (5% chance)
	// In a real implementation, this would call a payment gateway
	customerID := fmt.Sprintf("%v", customerId)
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

// ShipPackageActivity simulates shipping a package
func ShipPackageActivity(ctx context.Context, chargeResult ChargeResult) (any, error) {
	log.Printf("ship activity")

	// Simulate some processing time
	time.Sleep(3 * time.Second)

	// In a real implementation, this would call a shipping service
	result := ShipResult{
		TrackingID:        fmt.Sprintf("track_%d", time.Now().UnixNano()),
		Carrier:           "FastShip Express",
		EstimatedDelivery: time.Now().Add(3 * 24 * time.Hour), // 3 days from now
	}

	return result, nil
}

// DelayedActivity is used in the timer workflow example
func DelayedActivity(ctx context.Context, input any) (any, error) {
	time.Sleep(1 * time.Second)

	result := map[string]any{
		"message":   "Delayed activity completed",
		"timestamp": time.Now(),
		"input":     input,
	}

	return result, nil
}

// AddActivity is a simple activity for testing
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

	workerType := flag.String("worker", "both", "Type of worker to run: 'workflow', 'activity', or 'both'")
	flag.Parse()

	ctx := context.Background()
	nc, _ := nats.Connect("nats://localhost:4222", nil)
	workflowClient, err := client.NewClient(&client.Options{
		Conn:      nc,
		Namespace: "dev",
	})
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	// Start workers based on flag
	switch *workerType {
	case "workflow":
		runWorkflowWorker(workflowClient)
	case "activity":
		runActivityWorker(workflowClient)
	case "client":
		{
			future, err := workflowClient.ExecuteWorkflow(ctx, OrderWorkflow,
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
		}

	default:
		log.Fatalf("Invalid worker type: %s. Use 'workflow', 'activity', or 'client'", *workerType)
	}

}

func runWorkflowWorker(c client.Client) {
	ctx := context.Background()
	workerClient, err := worker.NewWorker(c, nil)
	if err != nil {
		log.Printf("err: %v", err)
		return
	}
	log.Println("Registering Workflow")
	err = workerClient.RegisterWorkflow(OrderWorkflow)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}

	if err := workerClient.Run(ctx); err != nil {
		log.Printf("err: %v", err)

		return
	}

}

func runActivityWorker(c client.Client) {
	ctx := context.Background()
	workerClient, err := worker.NewWorker(c, &worker.Options{
		Namespace: "dev",
	})
	if err != nil {
		log.Printf("err: %v", err)
		return
	}

	log.Println("Registering Activities")
	err = workerClient.RegisterActivity(AddActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}
	err = workerClient.RegisterActivity(DelayedActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}
	err = workerClient.RegisterActivity(ChargeCreditCardActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}
	err = workerClient.RegisterActivity(ShipPackageActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}

	if err := workerClient.Run(ctx); err != nil {
		log.Printf("err: %v", err)

		return
	}

}
