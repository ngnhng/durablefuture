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

package orderretries

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/ngnhng/durablefuture/examples/scenarios"
	"github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/worker"
	"github.com/ngnhng/durablefuture/sdk/workflow"
)

func init() {
	scenarios.Register(Example{})
}

// Example demonstrates automatic activity retries using retry policies.
// Unlike order-recovery which uses manual retry loops, this example uses
// declarative retry policies that the platform handles automatically.
type Example struct{}

func (Example) Name() string { return "order-retries" }

func (Example) RegisterWorkflows(reg worker.WorkflowRegistry) error {
	return reg.RegisterWorkflow(OrderWithRetriesWorkflow)
}

func (Example) RegisterActivities(reg worker.ActivityRegistry) error {
	activities := []any{
		TransientFailurePaymentActivity,
		NonRetryablePaymentActivity,
		EventuallySuccessfulShippingActivity,
	}
	for _, fn := range activities {
		if err := reg.RegisterActivity(fn); err != nil {
			return fmt.Errorf("register activity: %w", err)
		}
	}
	return nil
}

func (Example) RunClient(ctx context.Context, c client.Client) error {
	orderID := fmt.Sprintf("retry-demo-%d", time.Now().UnixNano())

	log.Println("\n=== Order Retries Demo ===")
	log.Println("This example demonstrates automatic activity retry policies.")
	log.Println("Watch the logs to see activities retrying with exponential backoff!")

	future, err := c.ExecuteWorkflow(ctx, OrderWithRetriesWorkflow,
		orderID,
		"demo-customer",
		299.99,
	)
	if err != nil {
		return fmt.Errorf("starting workflow failed: %w", err)
	}

	var result any
	if err := future.Get(ctx, &result); err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("\n=== Workflow Completed Successfully ===")
	log.Printf("Result: %+v\n", result)
	return nil
}

type (
	PaymentRequest struct {
		OrderID    string  `json:"order_id"`
		CustomerID string  `json:"customer_id"`
		Amount     float64 `json:"amount"`
	}

	PaymentResult struct {
		ChargeID    string    `json:"charge_id"`
		Amount      float64   `json:"amount"`
		AttemptNum  int32     `json:"attempt_num"`
		ProcessedAt time.Time `json:"processed_at"`
	}

	ShippingRequest struct {
		OrderID string        `json:"order_id"`
		Payment PaymentResult `json:"payment"`
	}

	ShippingResult struct {
		TrackingID        string    `json:"tracking_id"`
		Carrier           string    `json:"carrier"`
		EstimatedDelivery time.Time `json:"estimated_delivery"`
		AttemptNum        int32     `json:"attempt_num"`
	}
)

// OrderWithRetriesWorkflow demonstrates automatic activity retries using retry policies.
// Notice how clean the workflow code is - no manual retry loops needed!
func OrderWithRetriesWorkflow(ctx workflow.Context, orderID string, customerID string, amount float64) (any, error) {
	log.Printf("🚀 Starting order workflow: %s", orderID)

	// Configure retry policy for payment activity
	// This activity will fail twice before succeeding, demonstrating automatic retries
	paymentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,  // Total time budget across all retries
		StartToCloseTimeout:    30 * time.Second, // Time per individual attempt
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    2 * time.Second,  // Start with 2 second delay
			BackoffCoefficient: 2.0,              // Double the delay each retry
			MaximumInterval:    30 * time.Second, // Cap backoff at 30 seconds
			MaximumAttempts:    5,                // Allow up to 5 attempts
			NonRetryableErrorTypes: []string{
				"insufficient funds",
				"invalid card",
			},
		},
	})

	var payment PaymentResult
	if err := workflow.ExecuteActivity(
		paymentCtx,
		TransientFailurePaymentActivity,
		PaymentRequest{
			OrderID:    orderID,
			CustomerID: customerID,
			Amount:     amount,
		},
	).Get(ctx, &payment); err != nil {
		return nil, fmt.Errorf("payment failed after retries: %w", err)
	}

	log.Printf("💳 Payment successful after %d attempt(s): %s", payment.AttemptNum, payment.ChargeID)

	// Configure retry policy for shipping activity
	// This demonstrates a different retry configuration - more aggressive retries
	shippingCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    60 * time.Second,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1.5, // More gradual backoff
			MaximumInterval:    time.Minute,
			MaximumAttempts:    10, // More attempts for critical operation
			NonRetryableErrorTypes: []string{
				"address invalid",
				"product out of stock",
			},
		},
	})

	var shipping ShippingResult
	if err := workflow.ExecuteActivity(
		shippingCtx,
		EventuallySuccessfulShippingActivity,
		ShippingRequest{
			OrderID: orderID,
			Payment: payment,
		},
	).Get(ctx, &shipping); err != nil {
		return nil, fmt.Errorf("shipping failed after retries: %w", err)
	}

	log.Printf("📦 Shipping successful after %d attempt(s): %s", shipping.AttemptNum, shipping.TrackingID)

	return map[string]any{
		"order_id":           orderID,
		"charge_id":          payment.ChargeID,
		"amount":             payment.Amount,
		"payment_attempts":   payment.AttemptNum,
		"tracking_id":        shipping.TrackingID,
		"carrier":            shipping.Carrier,
		"shipping_attempts":  shipping.AttemptNum,
		"estimated_delivery": shipping.EstimatedDelivery,
		"status":             "completed",
	}, nil
}

// Global counters to track attempts (in production, use proper state management)
var (
	paymentAttempts  atomic.Int32
	shippingAttempts atomic.Int32
)

// TransientFailurePaymentActivity simulates a payment service with transient failures.
// It will fail the first 2 attempts, then succeed on the 3rd attempt.
// This demonstrates how retry policies handle transient errors automatically.
func TransientFailurePaymentActivity(ctx context.Context, req PaymentRequest) (PaymentResult, error) {
	attemptNum := paymentAttempts.Add(1)

	log.Printf("💳 [Attempt %d] Processing payment for order %s (amount: $%.2f)",
		attemptNum, req.OrderID, req.Amount)

	// Simulate transient failures for first 2 attempts
	if attemptNum <= 2 {
		time.Sleep(500 * time.Millisecond)
		log.Printf("❌ [Attempt %d] Payment failed: temporary payment gateway timeout", attemptNum)
		return PaymentResult{}, fmt.Errorf("payment gateway timeout (transient)")
	}

	// Success on 3rd attempt
	time.Sleep(time.Second)
	chargeID := fmt.Sprintf("ch_%d_%d", time.Now().UnixNano(), attemptNum)

	log.Printf("✅ [Attempt %d] Payment successful: %s", attemptNum, chargeID)

	return PaymentResult{
		ChargeID:    chargeID,
		Amount:      req.Amount,
		AttemptNum:  attemptNum,
		ProcessedAt: time.Now(),
	}, nil
}

// NonRetryablePaymentActivity demonstrates a non-retryable error.
// When the error message matches NonRetryableErrorTypes, no retries occur.
func NonRetryablePaymentActivity(ctx context.Context, req PaymentRequest) (PaymentResult, error) {
	log.Printf("💳 Processing payment for order %s", req.OrderID)

	// Simulate a permanent failure that shouldn't be retried
	if req.Amount < 0 {
		return PaymentResult{}, fmt.Errorf("invalid card")
	}

	return PaymentResult{
		ChargeID:    fmt.Sprintf("ch_%d", time.Now().UnixNano()),
		Amount:      req.Amount,
		AttemptNum:  1,
		ProcessedAt: time.Now(),
	}, nil
}

// EventuallySuccessfulShippingActivity simulates a shipping service that takes
// a few attempts to process the order successfully.
func EventuallySuccessfulShippingActivity(ctx context.Context, req ShippingRequest) (ShippingResult, error) {
	attemptNum := shippingAttempts.Add(1)

	log.Printf("📦 [Attempt %d] Processing shipment for order %s", attemptNum, req.OrderID)

	// Fail first 3 attempts with different errors
	switch attemptNum {
	case 1:
		time.Sleep(300 * time.Millisecond)
		log.Printf("❌ [Attempt %d] Shipping failed: warehouse system unavailable", attemptNum)
		return ShippingResult{}, fmt.Errorf("warehouse system unavailable")
	case 2:
		time.Sleep(400 * time.Millisecond)
		log.Printf("❌ [Attempt %d] Shipping failed: carrier API timeout", attemptNum)
		return ShippingResult{}, fmt.Errorf("carrier API timeout")
	case 3:
		time.Sleep(500 * time.Millisecond)
		log.Printf("❌ [Attempt %d] Shipping failed: temporary label printer error", attemptNum)
		return ShippingResult{}, fmt.Errorf("label printer error")
	}

	// Success on 4th+ attempt
	time.Sleep(time.Second)
	trackingID := fmt.Sprintf("TRK%d", time.Now().UnixNano())

	log.Printf("✅ [Attempt %d] Shipping successful: %s", attemptNum, trackingID)

	return ShippingResult{
		TrackingID:        trackingID,
		Carrier:           "Reliable Shipping Co",
		EstimatedDelivery: time.Now().Add(3 * 24 * time.Hour),
		AttemptNum:        attemptNum,
	}, nil
}
