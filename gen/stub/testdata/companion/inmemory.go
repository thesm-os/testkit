// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package companion

import (
	"context"
	"sync"
)

// InMemoryStore is a hand-written in-memory implementation of [Store].
// In production codebases, this would be the consumer's test companion —
// a lightweight implementation used for unit and integration tests.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string]string
}

// NewInMemoryStore returns a ready-to-use [InMemoryStore].
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]string)}
}

func (s *InMemoryStore) Get(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *InMemoryStore) Put(_ context.Context, key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *InMemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return ErrNotFound
	}
	delete(s.data, key)
	return nil
}
