// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readers

import (
	"context"
	"iter"
	"sync"
)

// InMemoryRegistry implements [Registry] for spec testing.
type InMemoryRegistry struct {
	mu       sync.Mutex
	handlers map[string]Handler
}

// NewInMemoryRegistry returns a ready-to-use [InMemoryRegistry].
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{handlers: make(map[string]Handler)}
}

// Register adds a handler to the registry.
func (r *InMemoryRegistry) Register(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[h.Name] = h
}

func (r *InMemoryRegistry) Lookup(ctx context.Context, name string) (Handler, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Handler{}, err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.handlers[name]
	if !ok {
		return Handler{}, ErrNotRegistered
	}
	return h, nil
}

func (r *InMemoryRegistry) List(ctx context.Context) iter.Seq2[Handler, error] {
	return func(yield func(Handler, error) bool) {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, h := range r.handlers {
			if ctx != nil && ctx.Err() != nil {
				yield(Handler{}, ctx.Err())
				return
			}
			if !yield(h, nil) {
				return
			}
		}
	}
}

func (r *InMemoryRegistry) Count(ctx context.Context) (int, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.handlers), nil
}
