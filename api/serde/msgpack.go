package serde

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

var _ BinarySerde = (*MsgpackSerde)(nil)

// MsgpackSerde implements the BinarySerde interface using MessagePack.
// MessagePack is a binary serialization format that is more efficient than JSON
// and preserves more type information (e.g., distinguishes between int and float).
type MsgpackSerde struct{}

// SerializeBinary serializes a Go value to MessagePack binary format.
func (m *MsgpackSerde) SerializeBinary(value any) ([]byte, error) {
	data, err := msgpack.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("msgpack serialization failed: %w", err)
	}
	return data, nil
}

// DeserializeBinary deserializes MessagePack binary data into a Go value.
func (m *MsgpackSerde) DeserializeBinary(data []byte, valuePtr any) error {
	if err := msgpack.Unmarshal(data, valuePtr); err != nil {
		return fmt.Errorf("msgpack deserialization failed: %w", err)
	}
	return nil
}
