// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

import (
	"context"
	"fmt"
	"sync"
)

// InMemoryDirectives is the [Directives] companion. Tracks per-method
// state so DelegateTo tests can assert calls flow through to the
// inner impl.
type InMemoryDirectives struct {
	mu sync.Mutex

	opened bool
	closed bool

	items map[string]Record

	// retryAttempt tracks the attempt counter for [Retry] so tests
	// can assert the stub's retry-succeeds sequencing matches the
	// directive's third-call success contract.
	retryAttempt map[string]int
}

// NewInMemoryDirectives returns an empty companion.
func NewInMemoryDirectives() *InMemoryDirectives {
	return &InMemoryDirectives{
		items:        make(map[string]Record),
		retryAttempt: make(map[string]int),
	}
}

// Seed prepopulates the items map.
func (d *InMemoryDirectives) Seed(items ...Record) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, it := range items {
		d.items[it.ID] = it
	}
}

// Opened reports whether Open has been called — used by stub
// auto-tests to verify order-after's prerequisite check.
func (d *InMemoryDirectives) Opened() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opened
}

// Open implements [Directives].
func (d *InMemoryDirectives) Open(_ context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.opened = true
	return nil
}

// Close implements [Directives]. (integration-only — but the
// companion still maintains real state so tests that bypass the
// stub can verify behavior.)
func (d *InMemoryDirectives) Close(_ context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

// Submit implements [Directives]. Returns ErrConflict when the
// item.ID already exists; otherwise stores and returns nil.
func (d *InMemoryDirectives) Submit(_ context.Context, item Record) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.items[item.ID]; exists {
		return ErrConflict
	}
	d.items[item.ID] = item
	return nil
}

// Wrap implements [Directives]. Always returns an error wrapped
// via ErrInternal — exercises the wrapped-via contract from the
// companion side.
func (d *InMemoryDirectives) Wrap(_ context.Context, key string) error {
	return fmt.Errorf("wrap %q: %w", key, ErrInternal)
}

// Legacy implements [Directives]. (deprecated — but the companion
// still works; the stub's Deprecated annotation is purely
// documentation.)
func (d *InMemoryDirectives) Legacy(ctx context.Context, item Record) error {
	return d.Submit(ctx, item)
}

// Retry implements [Directives]. Returns ErrInternal on attempts 1
// and 2; succeeds on attempt 3 onward. The companion's behavior
// matches the directive's contract so DelegateTo tests don't need
// the stub's retry sequencer to override.
func (d *InMemoryDirectives) Retry(_ context.Context, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.retryAttempt[key]++
	if d.retryAttempt[key] < 3 {
		return ErrInternal
	}
	return nil
}

// Read implements [Directives]. Returns ErrNotFound on miss.
func (d *InMemoryDirectives) Read(_ context.Context, key string) (Record, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, ok := d.items[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	return v, nil
}

// Shard implements [Directives].
func (d *InMemoryDirectives) Shard(_ context.Context, item Record) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items[item.ID] = item
	return nil
}

// ShardByKey implements [Directives].
func (d *InMemoryDirectives) ShardByKey(_ context.Context, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items[key] = Record{ID: key}
	return nil
}
