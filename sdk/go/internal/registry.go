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

package internal

import (
	"fmt"
	"reflect"
	"sync"
)

type registry interface {
	get(k string) (any, error)
	set(k string, v any) error
	size() int64
}

func NewInMemoryRegistry() registry {
	return newInMemoryRegistry()
}

func newInMemoryRegistry() *MapRegistry {
	return &MapRegistry{
		entries: make(map[string]any),
	}
}

type MapRegistry struct {
	mu      sync.RWMutex
	entries map[string]any
}

// Get retrieves the value for the key. No mutex since MapRegistry is only initialized during init.
func (m *MapRegistry) get(k string) (any, error) {
	entry, ok := m.entries[k]
	if !ok {
		return nil, fmt.Errorf("key %v have no value", k)
	}

	return entry, nil
}

func (m *MapRegistry) set(k string, v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.entries[k]
	if ok {
		return fmt.Errorf("key %v have value: %v", k, v)
	}

	fnType := reflect.TypeOf(v)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("entry '%s' is not a function", k)
	}

	m.entries[k] = v

	return nil
}

func (m *MapRegistry) size() int64 {
	return int64(len(m.entries))
}
