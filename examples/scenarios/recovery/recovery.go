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

package recovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ngnhng/durablefuture/examples/scenarios"
	"github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/worker"
	"github.com/ngnhng/durablefuture/sdk/workflow"
)

func init() {
	scenarios.Register(Example{})
}

// Example implements a workflow that intentionally fails the first activity attempt to showcase recovery.
type Example struct{}

func (Example) Name() string { return "order-recovery" }

func (Example) RegisterWorkflows(reg worker.WorkflowRegistry) error {
	return reg.RegisterWorkflow(RecoverableOrderWorkflow)
}

func (Example) RegisterActivities(reg worker.ActivityRegistry) error {
	if err := reg.RegisterActivity(FlakyChargeCreditCardActivity); err != nil {
		return fmt.Errorf("register payment activity: %w", err)
	}
	if err := reg.RegisterActivity(ShipAfterPaymentActivity); err != nil {
		return fmt.Errorf("register shipping activity: %w", err)
	}
	return nil
}

func (Example) RunClient(ctx context.Context, c client.Client) error {
	orderID := fmt.Sprintf("recovery-%d", time.Now().UnixNano())
	future, err := c.ExecuteWorkflow(ctx, RecoverableOrderWorkflow,
		orderID,
		"Bob",
		1000.0,
	)
	if err != nil {
		return fmt.Errorf("starting recovery workflow failed: %w", err)
	}

	var result any
	if err := future.Get(ctx, &result); err != nil {
		return fmt.Errorf("waiting for recovery workflow result failed: %w", err)
	}

	log.Printf("recovery workflow result: %v", result)
	return nil
}

type (
	ChargeRequest struct {
		OrderID    string
		CustomerID string
		Amount     float64
	}

	ChargeResult struct {
		OrderID string  `json:"order_id"`
		Amount  float64 `json:"amount"`
		Attempt int     `json:"attempt"`
		Charge  string  `json:"charge"`
	}

	ShipmentRequest struct {
		OrderID string       `json:"order_id"`
		Charge  ChargeResult `json:"charge"`
	}

	ShipResult struct {
		TrackingID string    `json:"tracking_id"`
		Carrier    string    `json:"carrier"`
		ETA        time.Time `json:"eta"`
	}
)

var failureGuard sync.Map

// RecoverableOrderWorkflow demonstrates a workflow that can recover from transient failures.
func RecoverableOrderWorkflow(ctx workflow.Context, orderID string, customerID string, amount float64) (any, error) {
	log.Printf("recoverable workflow started: order=%s", orderID)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
	})

	var (
		charge ChargeResult
		err    error
	)
	req := ChargeRequest{
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     amount,
	}

	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("attempting payment (attempt %d)", attempt)
		err = workflow.ExecuteActivity(ctx, FlakyChargeCreditCardActivity, req).Get(ctx, &charge)
		if err == nil {
			log.Printf("payment captured on attempt %d", attempt)
			break
		}
		log.Printf("payment attempt %d failed: %v", attempt, err)
		if attempt == 3 {
			return nil, fmt.Errorf("exhausted payment retries: %w", err)
		}
	}

	var shipment ShipResult
	if err := workflow.ExecuteActivity(ctx, ShipAfterPaymentActivity, ShipmentRequest{
		OrderID: orderID,
		Charge:  charge,
	}).Get(ctx, &shipment); err != nil {
		return nil, fmt.Errorf("shipping failed: %w", err)
	}

	return map[string]any{
		"order_id":      orderID,
		"charge_id":     charge.Charge,
		"amount":        charge.Amount,
		"tracking_id":   shipment.TrackingID,
		"carrier":       shipment.Carrier,
		"eta":           shipment.ETA,
		"retry_attempt": charge.Attempt,
	}, nil
}

// FlakyChargeCreditCardActivity intentionally fails the first time for each order to simulate a transient outage.
func FlakyChargeCreditCardActivity(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	if _, loaded := failureGuard.LoadOrStore(req.OrderID, struct{}{}); !loaded {
		log.Printf("simulating payment outage for order %s", req.OrderID)
		return ChargeResult{}, fmt.Errorf("temporary payment processor outage for %s", req.OrderID)
	}

	return ChargeResult{
		OrderID: req.OrderID,
		Amount:  req.Amount,
		Attempt: 2,
		Charge:  fmt.Sprintf("charge_%d", time.Now().UnixNano()),
	}, nil
}

func ShipAfterPaymentActivity(ctx context.Context, req ShipmentRequest) (ShipResult, error) {
	log.Printf("shipping order %s after charge %s", req.OrderID, req.Charge.Charge)
	time.Sleep(2 * time.Second)
	return ShipResult{
		TrackingID: fmt.Sprintf("track_%d", time.Now().UnixNano()),
		Carrier:    "Recovery Express",
		ETA:        time.Now().Add(48 * time.Hour),
	}, nil
}
