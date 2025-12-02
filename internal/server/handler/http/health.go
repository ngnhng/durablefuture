package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	conn      *jetstreamx.Connection
	startTime time.Time
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(conn *jetstreamx.Connection) *HealthHandler {
	return &HealthHandler{
		conn:      conn,
		startTime: time.Now(),
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}

// Health returns basic health status (always returns 200 if server is running)
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
		Checks:    make(map[string]string),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}

// Ready checks if the service is ready to accept traffic
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)
	checks := make(map[string]string)
	ready := true

	// Check NATS connection
	if h.conn == nil || !h.conn.IsConnected() {
		checks["nats"] = "disconnected"
		ready = false
	} else {
		checks["nats"] = "connected"
	}

	status := "ready"
	statusCode := http.StatusOK
	if !ready {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode ready response", "error", err)
	}
}
