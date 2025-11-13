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
