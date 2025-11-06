PROJECT_NAME=durable-future-demo

default: help

.PHONY: help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Workflow System:"
	@echo "  build-workflow      - Build all workflow components"
	@echo "  run-manager         - Run the workflow manager"
	@echo "  run-workflow-worker - Run a workflow worker"
	@echo "  run-activity-worker - Run an activity worker"
	@echo "  run-workflow-client - Run the workflow client"
	@echo "  start-workflow-demo - Start all workflow components (requires tmux)"
	@echo ""
	@echo "Development:"
	@echo "  test       - Run all tests"
	@echo "  fmt        - Format code"
	@echo "  mod-tidy   - Run go mod tidy"
	@echo "  clean      - Clean build artifacts"

.PHONY: build
build:
	go build ./...

.PHONY: build-workflow

.PHONY: run-worker
run-worker:
	go run ./worker/main.go

.PHONY: run-client
run-client:
	go run ./client/main.go

.PHONY: run-manager
run-manager:
	@echo "Starting Workflow Manager..."
	go run ./cmd/manager/main.go

.PHONY: run-workflow-worker

.PHONY: logs
logs:
	docker compose logs

.PHONY: test
test:
	go test ./... -v

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: mod-tidy
mod-tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin/
	go clean ./...
