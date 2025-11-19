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

import (
	"fmt"
	"reflect"
	"runtime"
)

// extractFullFunctionName extracts the function's name with the preceding packages details.
func extractFullFunctionName(fn any) (string, error) {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return "", fmt.Errorf("fn is not of function type")
	}
	fnObj := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if fnObj == nil {
		return "", fmt.Errorf("could not retrieve function metadata")
	}

	return fnObj.Name(), nil
}

// debugAnyValues returns a string representation of []any for debugging
func debugAnyValues(vals []any) string {
	if len(vals) == 0 {
		return "[]any{} (empty)"
	}

	result := fmt.Sprintf("[]any{%d items}:\n", len(vals))
	for i, v := range vals {
		if v == nil {
			result += fmt.Sprintf("  [%d]: <nil>\n", i)
		} else {
			t := reflect.TypeOf(v)
			result += fmt.Sprintf("  [%d]: %v (%v) = %+v\n", i, t, t.Kind(), v)
		}
	}
	return result
}
