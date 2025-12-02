package serde

import (
	"fmt"
	"reflect"
)

// TypeConverter provides serialization-agnostic type conversion utilities.
// It uses the provided BinarySerde to convert between types without assuming JSON semantics.
type TypeConverter struct {
	serde BinarySerde
}

// NewTypeConverter creates a new type converter using the provided serializer.
func NewTypeConverter(s BinarySerde) *TypeConverter {
	return &TypeConverter{serde: s}
}

// ConvertToType converts a value to the target type using serialization round-tripping.
// This approach is serializer-agnostic and handles complex type conversions.
func (tc *TypeConverter) ConvertToType(value any, targetType reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(targetType), nil
	}

	valueType := reflect.TypeOf(value)

	// If types already match, return as-is
	if valueType == targetType {
		return reflect.ValueOf(value), nil
	}

	// Handle direct conversions where possible (faster path)
	if valueType.ConvertibleTo(targetType) {
		// Special case: numeric conversions that might lose precision
		if isNumericKind(valueType.Kind()) && isNumericKind(targetType.Kind()) {
			return tc.convertNumeric(value, valueType, targetType)
		}
		return reflect.ValueOf(value).Convert(targetType), nil
	}

	// For complex types (structs, maps), use serialization round-tripping
	// This works regardless of the underlying serializer (JSON, msgpack, protobuf)
	return tc.convertViaSerializer(value, targetType)
}

// convertNumeric handles numeric type conversions with precision checking.
func (tc *TypeConverter) convertNumeric(value any, valueType, targetType reflect.Type) (reflect.Value, error) {
	// Handle float to int conversion (common after JSON deserialization)
	if valueType.Kind() == reflect.Float64 || valueType.Kind() == reflect.Float32 {
		if isIntegerKind(targetType.Kind()) {
			floatVal := reflect.ValueOf(value).Float()
			intVal := int64(floatVal)
			// Check for precision loss
			if float64(intVal) != floatVal {
				return reflect.Value{}, fmt.Errorf("cannot convert %v to %v without losing precision", floatVal, targetType)
			}
			// Convert to the specific integer type
			return reflect.ValueOf(intVal).Convert(targetType), nil
		}
	}

	// Handle int to float conversion (always safe)
	if isIntegerKind(valueType.Kind()) {
		if targetType.Kind() == reflect.Float64 || targetType.Kind() == reflect.Float32 {
			return reflect.ValueOf(value).Convert(targetType), nil
		}
	}

	// For other numeric conversions, use standard conversion
	if valueType.ConvertibleTo(targetType) {
		return reflect.ValueOf(value).Convert(targetType), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %v (%v) to %v", value, valueType, targetType)
}

// convertViaSerializer uses serialization round-tripping for complex type conversions.
// This is the key to serializer independence: we don't assume JSON semantics.
func (tc *TypeConverter) convertViaSerializer(value any, targetType reflect.Type) (reflect.Value, error) {
	// Serialize the value using the configured serializer
	data, err := tc.serde.SerializeBinary(value)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("failed to serialize value for type conversion: %w", err)
	}

	// Create a new instance of the target type
	var targetValue reflect.Value
	if targetType.Kind() == reflect.Ptr {
		targetValue = reflect.New(targetType.Elem())
	} else {
		targetValue = reflect.New(targetType)
	}

	// Deserialize into the target type
	if err := tc.serde.DeserializeBinary(data, targetValue.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("failed to deserialize value to target type: %w", err)
	}

	// Return the dereferenced value if target is not a pointer
	if targetType.Kind() != reflect.Ptr {
		return targetValue.Elem(), nil
	}
	return targetValue, nil
}

// ConvertSlice converts a slice of any to a slice of values matching the target element type.
func (tc *TypeConverter) ConvertSlice(values []any, targetElemType reflect.Type) ([]reflect.Value, error) {
	result := make([]reflect.Value, len(values))
	for i, val := range values {
		converted, err := tc.ConvertToType(val, targetElemType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert element %d: %w", i, err)
		}
		result[i] = converted
	}
	return result, nil
}

// Helper functions

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func isIntegerKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}
