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
	"log/slog"
	"os"

	serverapp "github.com/ngnhng/durablefuture/internal/server/app"
)

func main() {
	var (
		natsHost = flag.String("host", "localhost", "NATS server host")
		natsPort = flag.String("port", "4222", "NATS server port")
	)
	flag.Parse()

	ctx := context.Background()
	if err := serverapp.Run(ctx, serverapp.Options{
		NATSHost: *natsHost,
		NATSPort: *natsPort,
	}); err != nil {
		slog.Error("manager exited with error", "error", err)
		os.Exit(1)
	}
}
