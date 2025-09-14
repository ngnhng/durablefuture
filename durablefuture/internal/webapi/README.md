# Frontend Web Application

This directory contains the web frontend for DurableFuture workflow visualization.

## Overview

The frontend provides a modern, responsive web interface for:
- Executing workflows with real-time parameter input
- Visualizing workflow status and progress
- Viewing workflow execution history and events
- Real-time monitoring with auto-refresh

## Architecture

- **ConnectRPC Integration**: Uses ConnectRPC over HTTP for communication with the backend
- **Modern UI**: Clean, responsive design with gradient backgrounds and smooth animations
- **Real-time Updates**: Automatic refresh every 5 seconds to show workflow progress
- **Interactive Forms**: Dynamic form generation based on workflow type

## Features

### Workflow Execution
- Select from available workflows (currently OrderWorkflow)
- Dynamic parameter forms based on workflow requirements
- Real-time execution feedback with success/error messages

### Workflow Visualization
- List all active and completed workflows
- Status indicators (Running, Completed, Failed)
- Detailed input/output display
- Execution timestamps

### Workflow History
- View detailed execution events for each workflow
- Event timeline with timestamps
- Activity-level tracking

## Access

When the server is running, access the web interface at:
```
http://localhost:8080
```

## API Endpoints

The frontend communicates with these ConnectRPC endpoints:
- `POST /api.workflow.v1.WorkflowService/GetWorkflows` - List workflows
- `POST /api.workflow.v1.WorkflowService/GetWorkflowHistory` - Get workflow history
- `POST /api.workflow.v1.WorkflowService/ExecuteWorkflow` - Execute new workflow

## Development

The frontend is a single-page application using vanilla JavaScript, embedded directly in the Go binary as static files. No additional build steps are required.