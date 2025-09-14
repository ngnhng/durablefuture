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

package webapi

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

//go:embed static/*
var staticFiles embed.FS

// Server represents the web API server
type Server struct {
	workflowServer *WorkflowServer
	httpServer     *http.Server
}

// NewServer creates a new web API server
func NewServer(ctx context.Context, addr string) (*Server, error) {
	workflowServer, err := NewWorkflowServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow server: %w", err)
	}

	mux := http.NewServeMux()

	// Add ConnectRPC handler
	path, handler := workflowServer.Handler()
	mux.Handle(path, handler)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("failed to create static file system: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// Add CORS support for ConnectRPC
	corsHandler := corsMiddleware(mux)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(corsHandler, &http2.Server{}),
	}

	return &Server{
		workflowServer: workflowServer,
		httpServer:     httpServer,
	}, nil
}

// Start starts the web server
func (s *Server) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting web server", "addr", s.httpServer.Addr)
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.InfoContext(ctx, "Shutting down web server")
		return s.httpServer.Shutdown(context.Background())
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// corsMiddleware adds CORS headers for browser compatibility
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}