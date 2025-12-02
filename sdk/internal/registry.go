package internal

import (
	"fmt"
	"reflect"
	"sync"
)

func newInMemoryRegistry() *hashMapRegistry {
	return &hashMapRegistry{
		entries: make(map[string]any),
	}
}

type hashMapRegistry struct {
	mu      sync.RWMutex
	entries map[string]any
}

// Get retrieves the value for the key. No mutex since MapRegistry is only initialized during init.
func (m *hashMapRegistry) get(k string) (any, error) {
	entry, ok := m.entries[k]
	if !ok {
		return nil, fmt.Errorf("key %v have no value", k)
	}

	return entry, nil
}

func (m *hashMapRegistry) set(k string, v any) error {
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

func (m *hashMapRegistry) size() int64 {
	return int64(len(m.entries))
}
