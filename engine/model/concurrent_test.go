// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

func TestConcurrentLinearizable(t *testing.T) {
	t.Parallel()
	// Correct mutex-guarded store must be linearizable under concurrent access.
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithConcurrent(model.ConcurrentConfig[storeIface]{
			Workers:      4,
			OpsPerWorker: 30,
			Model:        linearize.KV[string, item](errNotFound),
			Actions: []model.ConcurrentAction[storeIface]{
				linearize.ConcurrentReader("Get", keyGen, storeGet),
				linearize.ConcurrentWriter("Put", itemGen, storePut, itemKey),
				linearize.ConcurrentDeleter("Delete", keyGen, storeDel),
			},
		}),
	)
}

func TestConcurrentNotLinearizable(t *testing.T) {
	t.Parallel()
	// nonLinearizableStore is thread-safe (mutex-guarded) but not
	// linearizable (Get reads stale snapshot). Safe to run under -race.
	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		model.Assert(
			ft,
			func() storeIface { return newNonLinearizableStore() },
			model.WithConcurrent(model.ConcurrentConfig[storeIface]{
				Workers:      4,
				OpsPerWorker: 30,
				Model:        linearize.KV[string, item](errNotFound),
				Actions: []model.ConcurrentAction[storeIface]{
					linearize.ConcurrentReader("Get", keyGen, storeGet),
					linearize.ConcurrentWriter("Put", itemGen, storePut, itemKey),
					linearize.ConcurrentDeleter("Delete", keyGen, storeDel),
				},
			}),
		)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent negative test timed out")
	}
	if !ft.Failed() {
		t.Fatal("racy store should fail linearizability check")
	}
}

// nonLinearizableStore uses a mutex but has a bug: Get returns a
// stale snapshot taken at construction time, not the current value.
// This is thread-safe but not linearizable — a Put followed by Get
// returns the old value.
type nonLinearizableStore struct {
	mu       sync.Mutex
	data     map[string]item
	snapshot map[string]item // stale snapshot
}

func newNonLinearizableStore() *nonLinearizableStore {
	return &nonLinearizableStore{
		data:     make(map[string]item),
		snapshot: make(map[string]item),
	}
}

func (s *nonLinearizableStore) Get(_ context.Context, key string) (item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// BUG: reads from stale snapshot, not current data
	v, ok := s.snapshot[key]
	if !ok {
		return item{}, errNotFound
	}
	return v, nil
}

func (s *nonLinearizableStore) Put(_ context.Context, v item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[v.ID] = v
	// BUG: snapshot is NEVER updated — reads always return stale data
	return nil
}

func (s *nonLinearizableStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *nonLinearizableStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}
