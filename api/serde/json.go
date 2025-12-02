package serde

import (
	"encoding/json"
	"fmt"
)

var _ BinarySerde = (*JsonSerde)(nil)

// JsonSerde implements the BinarySerde interface using JSON.
type JsonSerde struct{}

// SerializeBinary serializes a Go value to its JSON representation.
func (j *JsonSerde) SerializeBinary(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("json serialization failed: %w", err)
	}
	return data, nil
}

// DeserializeBinary deserializes JSON data into a Go value.
func (j *JsonSerde) DeserializeBinary(data []byte, valuePtr any) error {
	if err := json.Unmarshal(data, valuePtr); err != nil {
		return fmt.Errorf("json deserialization failed: %w", err)
	}
	return nil
}
