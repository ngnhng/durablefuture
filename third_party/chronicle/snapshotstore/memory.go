package snapshotstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/DeluxeOwl/chronicle/aggregate"
	"github.com/DeluxeOwl/chronicle/serde"
)

var _ aggregate.SnapshotStore[aggregate.ID, aggregate.Snapshot[aggregate.ID]] = (*MemoryStore[aggregate.ID, aggregate.Snapshot[aggregate.ID]])(
	nil,
)

// MemoryStore provides a thread-safe, in-memory implementation of the aggregate.SnapshotStore interface.
// It is useful for testing, development, or applications where snapshot persistence is not required.
// Snapshots are stored as serialized byte slices in a map, keyed by the aggregate ID.
type MemoryStore[TID aggregate.ID, TS aggregate.Snapshot[TID]] struct {
	serde          serde.BinarySerde
	snapshots      map[string][]byte
	createSnapshot func() TS
	mu             sync.RWMutex
}

type MemoryStoreOption[TID aggregate.ID, TS aggregate.Snapshot[TID]] func(*MemoryStore[TID, TS])

func WithSerializer[TID aggregate.ID, TS aggregate.Snapshot[TID]](
	s serde.BinarySerde,
) MemoryStoreOption[TID, TS] {
	return func(store *MemoryStore[TID, TS]) {
		store.serde = s
	}
}

// NewMemoryStore creates and returns a new in-memory snapshot store.
// It requires a constructor function for the specific snapshot type, which is used
// to create new instances during deserialization. By default, it uses JSON for serialization.
//
// Usage:
//
//	// Assuming account.Snapshot is your snapshot type
//	accountSnapshotStore := snapshotstore.NewMemoryStore(
//		func() *account.Snapshot { return new(account.Snapshot) },
//	)
//
// Returns a pointer to a fully initialized MemoryStore.
func NewMemoryStore[TID aggregate.ID, TS aggregate.Snapshot[TID]](
	createSnapshot func() TS,
	opts ...MemoryStoreOption[TID, TS],
) *MemoryStore[TID, TS] {
	store := &MemoryStore[TID, TS]{
		mu:             sync.RWMutex{},
		snapshots:      make(map[string][]byte),
		createSnapshot: createSnapshot,
		serde:          serde.NewJSONBinary(),
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// SaveSnapshot serializes the provided snapshot using the configured serde and saves
// it to the in-memory map. The operation is thread-safe.
//
// Usage:
//
//	// snapshot is a valid snapshot instance
//	err := store.SaveSnapshot(ctx, snapshot)
//	if err != nil {
//	    // handle error
//	}
//
// Returns an error if the serialization fails or if the context is cancelled.
func (s *MemoryStore[TID, TS]) SaveSnapshot(ctx context.Context, snapshot TS) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	id := snapshot.ID().String()

	data, err := s.serde.SerializeBinary(snapshot)
	if err != nil {
		return fmt.Errorf("save snapshot: marshal: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[id] = data

	return nil
}

// GetSnapshot retrieves a snapshot by its aggregate ID, deserializes it, and returns it.
// The operation is thread-safe.
//
// Usage:
//
//	// assuming 'id' is a valid aggregate.ID
//	snapshot, found, err := store.GetSnapshot(ctx, id)
//	if err != nil {
//	    // handle storage or deserialization error
//	}
//	if !found {
//	    // handle case where no snapshot exists
//	}
//	// use snapshot
//
// Returns the deserialized snapshot, a boolean indicating if a snapshot was found for the
// given ID, and an error if one occurred during retrieval or deserialization.
func (s *MemoryStore[TID, TS]) GetSnapshot(ctx context.Context, aggregateID TID) (TS, bool, error) {
	var empty TS

	if err := ctx.Err(); err != nil {
		return empty, false, err
	}

	id := aggregateID.String()

	s.mu.RLock()
	data, ok := s.snapshots[id]
	s.mu.RUnlock()

	if !ok {
		return empty, false, nil
	}

	snapshot := s.createSnapshot()
	if err := s.serde.DeserializeBinary(data, snapshot); err != nil {
		return empty, false, fmt.Errorf("get snapshot: unmarshal: %w", err)
	}

	return snapshot, true, nil
}
