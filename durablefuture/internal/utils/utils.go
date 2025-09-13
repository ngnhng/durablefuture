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

package utils

import (
	"durablefuture/internal/types"
	"fmt"
	"reflect"
	"runtime"
)

// ExtractFullFunctionName extracts the function's name with the preceding packages details.
func ExtractFullFunctionName(fn any) (string, error) {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return "", fmt.Errorf("fn is not of function type")
	}
	fnObj := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if fnObj == nil {
		return "", fmt.Errorf("could not retrieve function metadata")
	}

	return fnObj.Name(), nil
}

func ReflectValuesToAny(vals []reflect.Value) []any {
	anySlice := make([]any, len(vals))
	for i, v := range vals {
		anySlice[i] = v.Interface()
	}
	return anySlice
}

// DebugReflectValues returns a string representation of []reflect.Value for debugging
func DebugReflectValues(vals []reflect.Value) string {
	if len(vals) == 0 {
		return "[]reflect.Value{} (empty)"
	}

	result := fmt.Sprintf("[]reflect.Value{%d items}:\n", len(vals))
	for i, v := range vals {
		if v.IsValid() {
			if v.CanInterface() {
				result += fmt.Sprintf("  [%d]: %v (%v) = %+v\n", i, v.Type(), v.Kind(), v.Interface())
			} else {
				result += fmt.Sprintf("  [%d]: %v (%v) = <unexportable>\n", i, v.Type(), v.Kind())
			}
		} else {
			result += fmt.Sprintf("  [%d]: <invalid reflect.Value>\n", i)
		}
	}
	return result
}

// DebugAnyValues returns a string representation of []any for debugging
func DebugAnyValues(vals []any) string {
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

// DebugWorkflowEvents returns a string representation of []types.WorkflowEvent for debugging
func DebugWorkflowEvents(events []types.WorkflowEvent) string {
	if len(events) == 0 {
		return "[]types.WorkflowEvent{} (empty)"
	}

	result := fmt.Sprintf("[]types.WorkflowEvent{%d items}:\n", len(events))
	for i, e := range events {
		result += fmt.Sprintf("  [%d]: EventID=%d, EventType=%d, WorkflowID=%s, Attributes=%s\n",
			i, e.EventID, e.EventType, e.WorkflowID, string(e.Attributes))
	}
	return result
}
