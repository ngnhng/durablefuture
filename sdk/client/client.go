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

package client

import "github.com/ngnhng/durablefuture/sdk/internal"

type (
	Client  = internal.Client
	Options = internal.ClientOptions
)

// NewClient creates a client using the provided Options.
func NewClient(options *Options) (Client, error) {
	return internal.NewClient(options)
}
