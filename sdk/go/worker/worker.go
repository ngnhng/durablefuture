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

package worker

import (
	"context"

	"github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/internal"
)

type (
	Worker interface {
		Registry
		Run(ctx context.Context) error
	}

	Registry interface {
		WorkflowRegistry
		ActivityRegistry
	}

	WorkflowRegistry = internal.WorkflowRegistry

	ActivityRegistry = internal.ActivityRegistry

	Options = internal.WorkerOptions
)

func NewWorker(c client.Client, options *Options) (Worker, error) {
	return internal.NewWorker(c, options)
}
