// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package richstruct

import (
	"context"
	"sync"
)

// InMemoryStore implements [Store] for model testing.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string]Document
}

// NewInMemoryStore returns a ready-to-use [InMemoryStore].
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Document)}
}

func (s *InMemoryStore) Get(_ context.Context, id string) (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return Document{}, ErrNotFound
	}
	return v, nil
}

func (s *InMemoryStore) Put(_ context.Context, doc Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[doc.ID] = doc
	return nil
}

func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *InMemoryStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}
