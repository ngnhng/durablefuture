// Copyright 2025 Nguyen-Nhat Nguyen
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
	"log"
	"time"

	"durablefuture/examples"
	"durablefuture/internal/types"
	"durablefuture/sdk/client"
	"durablefuture/sdk/worker"
)

func main() {
	log.Println("DurableFuture Configuration Examples")

	ctx := context.Background()

	// Example 1: Start workflow workers with retry examples
	go func() {
		workerClient, err := worker.NewWorker()
		if err != nil {
			log.Printf("Failed to create workflow worker: %v", err)
			return
		}
		
		// Register retry example workflows
		err = workerClient.RegisterWorkflow(examples.RetryExampleWorkflow)
		if err != nil {
			log.Printf("Failed to register RetryExampleWorkflow: %v", err)
			return
		}
		
		err = workerClient.RegisterWorkflow(examples.TimeoutExampleWorkflow)
		if err != nil {
			log.Printf("Failed to register TimeoutExampleWorkflow: %v", err)
			return
		}

		if err := workerClient.Run(ctx); err != nil {
			log.Printf("Workflow worker failed: %v", err)
		}
	}()

	// Example 2: Start activity workers with new activities
	go func() {
		time.Sleep(1 * time.Second) // Wait a bit for workflow worker to start
		
		activityWorker, err := worker.NewWorker()
		if err != nil {
			log.Printf("Failed to create activity worker: %v", err)
			return
		}

		// Register retry example activities
		err = activityWorker.RegisterActivity(examples.FlakyActivity)
		if err != nil {
			log.Printf("Failed to register FlakyActivity: %v", err)
			return
		}
		
		err = activityWorker.RegisterActivity(examples.ActivityWithNonRetryableError)
		if err != nil {
			log.Printf("Failed to register ActivityWithNonRetryableError: %v", err)
			return
		}
		
		err = activityWorker.RegisterActivity(examples.SlowActivity)
		if err != nil {
			log.Printf("Failed to register SlowActivity: %v", err)
			return
		}

		if err := activityWorker.Run(ctx); err != nil {
			log.Printf("Activity worker failed: %v", err)
		}
	}()

	// Example 3: Execute workflows with different configurations
	time.Sleep(3 * time.Second) // Wait for workers to start

	// Create workflow client
	workflowClient, err := client.NewClient()
	if err != nil {
		log.Fatalf("Failed to create workflow client: %v", err)
	}

	// Execute retry example workflow
	log.Println("=== Executing Retry Example Workflow ===")
	retryFuture, err := workflowClient.ExecuteWorkflow(ctx, examples.RetryExampleWorkflow, "test-input")
	if err != nil {
		log.Printf("Failed to start retry example workflow: %v", err)
	} else {
		var retryResult any
		err = retryFuture.Get(ctx, &retryResult)
		if err != nil {
			log.Printf("Retry example workflow failed: %v", err)
		} else {
			log.Printf("Retry example workflow result: %v", retryResult)
		}
	}

	// Execute timeout example workflow
	log.Println("\n=== Executing Timeout Example Workflow ===")
	timeoutFuture, err := workflowClient.ExecuteWorkflow(ctx, examples.TimeoutExampleWorkflow, "timeout-test")
	if err != nil {
		log.Printf("Failed to start timeout example workflow: %v", err)
	} else {
		var timeoutResult any
		err = timeoutFuture.Get(ctx, &timeoutResult)
		if err != nil {
			log.Printf("Timeout example workflow failed: %v", err)
		} else {
			log.Printf("Timeout example workflow result: %v", timeoutResult)
		}
	}

	// Example 4: Demonstrate namespace configuration
	log.Println("\n=== Namespace Configuration Example ===")
	productionNamespace := types.DefaultNamespaceConfig("production")
	log.Printf("Production namespace config: %+v", productionNamespace)

	stagingNamespace := &types.NamespaceConfig{
		Name:        "staging",
		Description: "Staging environment with relaxed limits",
		RetentionPolicy: &types.RetentionPolicy{
			WorkflowHistoryRetention:   7 * 24 * time.Hour,  // 7 days
			CompletedWorkflowRetention: 3 * 24 * time.Hour,  // 3 days
			FailedWorkflowRetention:    7 * 24 * time.Hour,  // 7 days
		},
		ResourceLimits: &types.ResourceLimits{
			MaxConcurrentWorkflows:   100,
			MaxWorkflowsPerSecond:    20,
			MaxActivityExecutionTime: 30 * time.Minute,
		},
	}
	log.Printf("Staging namespace config: %+v", stagingNamespace)

	// Validate namespaces
	if err := types.ValidateNamespace(productionNamespace.Name); err != nil {
		log.Printf("Production namespace validation failed: %v", err)
	} else {
		log.Printf("Production namespace '%s' is valid", productionNamespace.Name)
	}

	if err := types.ValidateNamespace(stagingNamespace.Name); err != nil {
		log.Printf("Staging namespace validation failed: %v", err)
	} else {
		log.Printf("Staging namespace '%s' is valid", stagingNamespace.Name)
	}

	// Example 5: Demonstrate retry policy configurations
	log.Println("\n=== Retry Policy Examples ===")
	
	// Exponential backoff
	exponentialPolicy := &types.RetryPolicy{
		MaximumAttempts:    5,
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    60 * time.Second,
		BackoffPolicy:      types.BackoffExponential,
	}
	log.Printf("Exponential backoff delays:")
	for i := 0; i < 4; i++ {
		delay := exponentialPolicy.CalculateRetryDelay(i)
		log.Printf("  Attempt %d: %v", i+1, delay)
	}

	// Linear backoff
	linearPolicy := &types.RetryPolicy{
		MaximumAttempts: 4,
		InitialInterval: 2 * time.Second,
		MaximumInterval: 30 * time.Second,
		BackoffPolicy:   types.BackoffLinear,
	}
	log.Printf("Linear backoff delays:")
	for i := 0; i < 4; i++ {
		delay := linearPolicy.CalculateRetryDelay(i)
		log.Printf("  Attempt %d: %v", i+1, delay)
	}

	// Fixed backoff
	fixedPolicy := &types.RetryPolicy{
		MaximumAttempts: 3,
		InitialInterval: 5 * time.Second,
		BackoffPolicy:   types.BackoffFixed,
	}
	log.Printf("Fixed backoff delays:")
	for i := 0; i < 3; i++ {
		delay := fixedPolicy.CalculateRetryDelay(i)
		log.Printf("  Attempt %d: %v", i+1, delay)
	}

	log.Println("\n=== Configuration Examples Complete ===")
}