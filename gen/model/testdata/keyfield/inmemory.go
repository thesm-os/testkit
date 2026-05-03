// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package keyfield

import (
	"context"
	"sync"
)

// InMemoryStore implements [Store] for model testing.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string]Record
}

// NewInMemoryStore returns a ready-to-use [InMemoryStore].
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Record)}
}

func (s *InMemoryStore) Get(_ context.Context, key string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	return v, nil
}

func (s *InMemoryStore) Put(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[record.Key] = record
	return nil
}

func (s *InMemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}
