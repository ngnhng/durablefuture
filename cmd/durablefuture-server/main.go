package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	serverapp "github.com/ngnhng/durablefuture/internal/server/app"
)

func main() {
	var (
		natsHost = flag.String("host", "localhost", "NATS server host")
		natsPort = flag.String("port", "4222", "NATS server port")
		httpPort = flag.String("http-port", "8080", "HTTP server port")
	)
	flag.Parse()

	ctx := context.Background()
	if err := serverapp.Run(ctx, serverapp.Options{
		NATSHost: *natsHost,
		NATSPort: *natsPort,
		HTTPPort: *httpPort,
	}); err != nil {
		slog.Error("manager exited with error", "error", err)
		os.Exit(1)
	}
}
