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

package workflow

import (
	"context"
)

type BlockingFutureError struct{}

func (e BlockingFutureError) Error() string {
	return "blocking_future"
}

// Future is an interface that represents the result of an asynchronous operation.
type Future interface {
	// Get blocks until the future is ready and returns the result.
	Get(ctx context.Context, valuePtr any) error
}
