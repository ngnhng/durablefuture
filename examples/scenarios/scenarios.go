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

package scenarios

import (
	"context"
	"sort"

	clientpkg "github.com/ngnhng/durablefuture/sdk/client"
	"github.com/ngnhng/durablefuture/sdk/worker"
)

// Example describes an executable workflow scenario that can register workflows,
// register activities, and run a client invocation.
type Example interface {
	Name() string
	RegisterWorkflows(worker.WorkflowRegistry) error
	RegisterActivities(worker.ActivityRegistry) error
	RunClient(ctx context.Context, c clientpkg.Client) error
}

var registry = map[string]Example{}

// Register adds an example implementation to the registry.
func Register(example Example) {
	if example == nil {
		return
	}
	registry[example.Name()] = example
}

// Get returns the example by name if it exists.
func Get(name string) (Example, bool) {
	example, ok := registry[name]
	return example, ok
}

// Names returns the list of registered example names in alphabetical order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
