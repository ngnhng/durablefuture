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

package http

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	handler *DemoHandler
	server  *http.Server
}

func NewServer(conn *jetstreamx.Connection, serde serde.BinarySerde, port string) *Server {
	handler := NewDemoHandler(conn, serde)

	mux := http.NewServeMux()

	// Demo API endpoints
	mux.HandleFunc("/api/demo/start", handler.StartWorkflow)
	mux.HandleFunc("/api/demo/status", handler.GetStatus)
	mux.HandleFunc("/api/demo/crash", handler.Crash)

	// Serve frontend
	frontendPath := os.Getenv("FRONTEND_BUILD_PATH")
	if frontendPath == "" {
		frontendPath = "./frontend/build/client"
	}

	// Check if production build exists
	if _, err := os.Stat(frontendPath); err == nil {
		slog.Info("Serving frontend from production build", "path", frontendPath)
		fileServer := http.FileServer(http.Dir(frontendPath))
		mux.Handle("/", fileServer)
	} else {
		// Serve embedded static files
		slog.Info("Serving frontend from embedded static files")
		staticContent, _ := fs.Sub(staticFS, "static")
		mux.Handle("/", http.FileServer(http.FS(staticContent)))
	}

	return &Server{
		handler: handler,
		server: &http.Server{
			Addr:    ":" + port,
			Handler: corsMiddleware(mux),
		},
	}
}

func (s *Server) Start(ctx context.Context) error {
	slog.Info("Starting HTTP server", "addr", s.server.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
