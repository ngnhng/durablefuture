package common

import (
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

func ReflectValuesToAny(vals []reflect.Value) []any {
	anySlice := make([]any, len(vals))
	for i, v := range vals {
		anySlice[i] = v.Interface()
	}
	return anySlice
}
