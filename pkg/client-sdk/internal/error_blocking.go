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

package internal

// ErrorBlockingFuture is thrown in a panic if a Future is not ready when Get is called.
// The concept is similar to the yield mechanism in coroutines, this will pause the workflow and resume it when the Future is ready.
type ErrorBlockingFuture struct{}

func (e ErrorBlockingFuture) Error() string {
	return "blocking_future"
}
