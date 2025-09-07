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

package converter

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
)

type JsonConverter struct{}

func NewJsonConverter() Converter {
	return &JsonConverter{}
}

func (c *JsonConverter) To(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}

	return json.Marshal(v)
}

func (c *JsonConverter) From(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("converter.From: must pass a non-nil pointer, not a %T", v)
	}

	return json.Unmarshal(data, v)
}

func (c *JsonConverter) MustTo(v any) []byte {
	if v == nil {
		return nil
	}

	if data, err := json.Marshal(v); err != nil {
		log.Printf("error MustTo: %v", err)
		panic(err)
	} else {
		return data
	}
}

func (c *JsonConverter) MustFrom(data []byte, v any) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		panic(fmt.Errorf("converter.MustFrom: must pass a non-nil pointer, not a %T", v))
	}

	if err := json.Unmarshal(data, v); err != nil {
		log.Printf("error MustFrom: failed to unmarshal into %T: %v", v, err)
		panic(err)
	}
}
