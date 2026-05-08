// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package allshapes

import (
	"context"
	"iter"
	"sync"
)

// InMemoryService implements [Service] for spec testing.
type InMemoryService struct {
	mu     sync.Mutex
	data   map[string]Item
	closed bool
	err    error
}

// NewInMemoryService returns a ready-to-use [InMemoryService].
func NewInMemoryService() *InMemoryService {
	return &InMemoryService{data: make(map[string]Item)}
}

func (s *InMemoryService) Get(ctx context.Context, id string) (Item, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Item{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (s *InMemoryService) Put(ctx context.Context, item Item) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	return nil
}

func (s *InMemoryService) Delete(ctx context.Context, id string) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

func (s *InMemoryService) Count(ctx context.Context) (int, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}

func (s *InMemoryService) Close(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *InMemoryService) Describe() string {
	return "in-memory service"
}

func (s *InMemoryService) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data) == 0
}

func (s *InMemoryService) List(ctx context.Context) iter.Seq2[Item, error] {
	return func(yield func(Item, error) bool) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, item := range s.data {
			if ctx != nil && ctx.Err() != nil {
				yield(Item{}, ctx.Err())
				return
			}
			if !yield(item, nil) {
				return
			}
		}
	}
}

func (s *InMemoryService) Touch(_ context.Context, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.data[id]; ok {
		s.data[id] = item
	}
}

func (s *InMemoryService) Load(_ context.Context, id string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.data[id]
	return item, ok
}

func (s *InMemoryService) Inspect(id string) (Item, Metadata, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.data[id]
	if !ok {
		return Item{}, Metadata{}, false
	}
	return item, Metadata{Version: "v1"}, true
}

func (s *InMemoryService) Err() error { return s.err }

func (s *InMemoryService) Merge(_ context.Context, id string, patch Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.data[id]; ok {
		if patch.Name != "" {
			existing.Name = patch.Name
		}
		s.data[id] = existing
	}
}
