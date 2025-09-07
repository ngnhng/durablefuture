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
	"fmt"
	"log/slog"
	"os"

	"durablefuture/internal/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	commandName := os.Args[1]
	commandArgs := os.Args[2:]

	var command cmd.Command
	ctx := context.Background()

	switch commandName {
	case "serve":
		command = cmd.NewServerCommand(ctx)
	default:
		printUsage()
		os.Exit(1)
	}

	if err := command.Execute(ctx, commandArgs); err != nil {
		slog.ErrorContext(ctx, "command failed", "error", err)
	}
}

func printUsage() {
	fmt.Printf("Usage: %s <command> [options]\n", os.Args[0])
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  serve    Start the DurableFuture manager server")
	fmt.Println()
	fmt.Println("\tServer options:")
	fmt.Println("\t  -host string    NATS server host (default \"localhost\")")
	fmt.Println("\t  -port string    NATS server port (default \"4222\")")
}
