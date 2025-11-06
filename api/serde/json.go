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
