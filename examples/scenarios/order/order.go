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

package order

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ngnhng/durablefuture/examples/scenarios"
	clientpkg "github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/worker"
	"github.com/ngnhng/durablefuture/sdk/workflow"
)

func init() {
	scenarios.Register(Example{})
}

// Example implements the classic order workflow scenario.
type Example struct{}

func (Example) Name() string { return "order" }

func (Example) RegisterWorkflows(reg worker.WorkflowRegistry) error {
	return reg.RegisterWorkflow(OrderWorkflow)
}

func (Example) RegisterActivities(reg worker.ActivityRegistry) error {
	for _, entry := range []struct {
		name string
		fn   any
	}{
		{"AddActivity", AddActivity},
		{"DelayedActivity", DelayedActivity},
		{"ChargeCreditCardActivity", ChargeCreditCardActivity},
		{"ShipPackageActivity", ShipPackageActivity},
	} {
		if err := reg.RegisterActivity(entry.fn); err != nil {
			return fmt.Errorf("register %s: %w", entry.name, err)
		}
	}
	return nil
}

func (Example) RunClient(ctx context.Context, c clientpkg.Client) error {
	future, err := c.ExecuteWorkflow(ctx, OrderWorkflow,
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

	log.Printf("order workflow result: %v", result)
	return nil
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
		"charged_amount":     chargeResult.Amount,
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

	time.Sleep(2 * time.Second)

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

	a, ok := inputMap["a"].(float64)
	if !ok {
		return nil, fmt.Errorf("input 'a' must be a number")
	}

	b, ok := inputMap["b"].(float64)
	if !ok {
		return nil, fmt.Errorf("input 'b' must be a number")
	}

	return map[string]any{
		"result": a + b,
	}, nil
}
