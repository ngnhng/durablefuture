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

package serde_test

import (
	"reflect"
	"testing"

	"github.com/ngnhng/durablefuture/api/serde"
)

// TestData represents a complex struct with various types
type TestData struct {
	Name    string         `json:"name" msgpack:"name"`
	Age     int            `json:"age" msgpack:"age"`
	Score   float64        `json:"score" msgpack:"score"`
	Active  bool           `json:"active" msgpack:"active"`
	Tags    []string       `json:"tags" msgpack:"tags"`
	Nested  *NestedData    `json:"nested,omitempty" msgpack:"nested,omitempty"`
	Mapping map[string]any `json:"mapping" msgpack:"mapping"`
}

type NestedData struct {
	Value string `json:"value" msgpack:"value"`
	Count int    `json:"count" msgpack:"count"`
}

// TestSerializationAgnostic verifies that TypeConverter works with different serializers
func TestSerializationAgnostic(t *testing.T) {
	testCases := []struct {
		name  string
		serde serde.BinarySerde
	}{
		{"JSON", &serde.JsonSerde{}},
		{"MessagePack", &serde.MsgpackSerde{}},
	}

	originalData := TestData{
		Name:   "Alice",
		Age:    30,
		Score:  95.5,
		Active: true,
		Tags:   []string{"tag1", "tag2", "tag3"},
		Nested: &NestedData{
			Value: "nested_value",
			Count: 42,
		},
		Mapping: map[string]any{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize
			serialized, err := tc.serde.SerializeBinary(originalData)
			if err != nil {
				t.Fatalf("Serialization failed: %v", err)
			}

			// Deserialize
			var deserialized TestData
			if err := tc.serde.DeserializeBinary(serialized, &deserialized); err != nil {
				t.Fatalf("Deserialization failed: %v", err)
			}

			// Verify
			if deserialized.Name != originalData.Name {
				t.Errorf("Name mismatch: got %v, want %v", deserialized.Name, originalData.Name)
			}
			if deserialized.Age != originalData.Age {
				t.Errorf("Age mismatch: got %v, want %v", deserialized.Age, originalData.Age)
			}
			if deserialized.Score != originalData.Score {
				t.Errorf("Score mismatch: got %v, want %v", deserialized.Score, originalData.Score)
			}
			if deserialized.Active != originalData.Active {
				t.Errorf("Active mismatch: got %v, want %v", deserialized.Active, originalData.Active)
			}
			if !reflect.DeepEqual(deserialized.Tags, originalData.Tags) {
				t.Errorf("Tags mismatch: got %v, want %v", deserialized.Tags, originalData.Tags)
			}
			if deserialized.Nested.Value != originalData.Nested.Value {
				t.Errorf("Nested.Value mismatch: got %v, want %v", deserialized.Nested.Value, originalData.Nested.Value)
			}
			if deserialized.Nested.Count != originalData.Nested.Count {
				t.Errorf("Nested.Count mismatch: got %v, want %v", deserialized.Nested.Count, originalData.Nested.Count)
			}
		})
	}
}

// TestTypeConverter verifies that TypeConverter preserves type information
func TestTypeConverter(t *testing.T) {
	testCases := []struct {
		name  string
		serde serde.BinarySerde
	}{
		{"JSON", &serde.JsonSerde{}},
		{"MessagePack", &serde.MsgpackSerde{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			converter := serde.NewTypeConverter(tc.serde)

			t.Run("IntConversion", func(t *testing.T) {
				// Simulate what happens when an int goes through serialization
				original := 42

				// Serialize and deserialize through any
				serialized, _ := tc.serde.SerializeBinary(original)
				var anyValue any
				tc.serde.DeserializeBinary(serialized, &anyValue)

				// Convert back to int using TypeConverter
				result, err := converter.ConvertToType(anyValue, reflect.TypeOf(0))
				if err != nil {
					t.Fatalf("Type conversion failed: %v", err)
				}

				if result.Interface() != original {
					t.Errorf("Int conversion failed: got %v (%T), want %v (%T)",
						result.Interface(), result.Interface(), original, original)
				}
			})

			t.Run("StructConversion", func(t *testing.T) {
				original := NestedData{
					Value: "test",
					Count: 99,
				}

				// Serialize to map[string]any (simulating intermediate representation)
				serialized, _ := tc.serde.SerializeBinary(original)
				var mapValue map[string]any
				tc.serde.DeserializeBinary(serialized, &mapValue)

				// Convert map back to struct using TypeConverter
				result, err := converter.ConvertToType(mapValue, reflect.TypeOf(NestedData{}))
				if err != nil {
					t.Fatalf("Struct conversion failed: %v", err)
				}

				converted := result.Interface().(NestedData)
				if converted.Value != original.Value || converted.Count != original.Count {
					t.Errorf("Struct conversion mismatch: got %+v, want %+v", converted, original)
				}
			})

			t.Run("SliceConversion", func(t *testing.T) {
				original := []string{"a", "b", "c"}

				// Serialize to []any
				serialized, _ := tc.serde.SerializeBinary(original)
				var anySlice []any
				tc.serde.DeserializeBinary(serialized, &anySlice)

				// Convert each element
				results, err := converter.ConvertSlice(anySlice, reflect.TypeOf(""))
				if err != nil {
					t.Fatalf("Slice conversion failed: %v", err)
				}

				if len(results) != len(original) {
					t.Fatalf("Slice length mismatch: got %d, want %d", len(results), len(original))
				}

				for i, result := range results {
					if result.Interface() != original[i] {
						t.Errorf("Slice element %d mismatch: got %v, want %v", i, result.Interface(), original[i])
					}
				}
			})
		})
	}
}

// TestNumericPrecision verifies that numeric type conversions handle precision correctly
func TestNumericPrecision(t *testing.T) {
	testCases := []struct {
		name        string
		serde       serde.BinarySerde
		expectFloat bool // JSON converts all numbers to float64
	}{
		{"JSON", &serde.JsonSerde{}, true},
		{"MessagePack", &serde.MsgpackSerde{}, false}, // msgpack preserves int vs float
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			converter := serde.NewTypeConverter(tc.serde)

			// Test integer value
			original := 42
			serialized, _ := tc.serde.SerializeBinary(original)
			var anyValue any
			tc.serde.DeserializeBinary(serialized, &anyValue)

			// Check intermediate type
			intermediateType := reflect.TypeOf(anyValue)
			if tc.expectFloat {
				if intermediateType.Kind() != reflect.Float64 {
					t.Logf("Note: %s represents integers as %v (expected float64)", tc.name, intermediateType.Kind())
				}
			} else {
				if intermediateType.Kind() == reflect.Float64 {
					t.Logf("Note: %s unexpectedly converted int to float64", tc.name)
				}
			}

			// TypeConverter should handle both cases
			result, err := converter.ConvertToType(anyValue, reflect.TypeOf(0))
			if err != nil {
				t.Fatalf("Type conversion failed: %v", err)
			}

			if result.Interface() != original {
				t.Errorf("Value mismatch: got %v, want %v", result.Interface(), original)
			}
		})
	}
}
