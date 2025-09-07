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
	"log"

	"durablefuture/examples"
	"durablefuture/sdk/client"
	"durablefuture/sdk/worker"
)

func main() {

	workerType := flag.String("worker", "both", "Type of worker to run: 'workflow', 'activity', or 'both'")
	flag.Parse()

	ctx := context.Background()
	workflowClient, err := client.NewClient()
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	// Start workers based on flag
	switch *workerType {
	case "workflow":
		runWorkflowWorker()
	case "activity":
		runActivityWorker()
	case "client":
		{
			future, err := workflowClient.ExecuteWorkflow(ctx, examples.OrderWorkflow,
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

func runWorkflowWorker() {
	ctx := context.Background()
	workerClient, err := worker.NewWorker()
	if err != nil {
		log.Printf("err: %v", err)
		return
	}
	log.Println("Registering Workflow")
	err = workerClient.RegisterWorkflow(examples.OrderWorkflow)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}

	if err := workerClient.Run(ctx); err != nil {
		log.Printf("err: %v", err)

		return
	}

}

func runActivityWorker() {
	ctx := context.Background()
	workerClient, err := worker.NewWorker()
	if err != nil {
		log.Printf("err: %v", err)
		return
	}

	log.Println("Registering Activities")
	err = workerClient.RegisterActivity(examples.AddActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}
	err = workerClient.RegisterActivity(examples.DelayedActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}
	err = workerClient.RegisterActivity(examples.ChargeCreditCardActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}
	err = workerClient.RegisterActivity(examples.ShipPackageActivity)
	if err != nil {
		log.Printf("err: %v", err)

		return
	}

	if err := workerClient.Run(ctx); err != nil {
		log.Printf("err: %v", err)

		return
	}

}
