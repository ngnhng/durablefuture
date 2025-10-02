package serde

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

var _ BinarySerde = (*ProtoSerde)(nil)

// ProtoSerde implements the BinarySerde interface using Protobuf.
type ProtoSerde struct{}

// SerializeBinary serializes a proto.Message to its binary representation.
func (p *ProtoSerde) SerializeBinary(value any) ([]byte, error) {
	msg, ok := value.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("value is not a proto.Message")
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("protobuf serialization failed: %w", err)
	}
	return data, nil
}

// DeserializeBinary deserializes binary data into a proto.Message.
func (p *ProtoSerde) DeserializeBinary(data []byte, valuePtr any) error {
	msg, ok := valuePtr.(proto.Message)
	if !ok {
		return fmt.Errorf("valuePtr is not a proto.Message")
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("protobuf deserialization failed: %w", err)
	}
	return nil
}
