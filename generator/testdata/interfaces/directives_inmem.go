// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

import (
	"context"
	"errors"
	"sync"
)

// InMemoryDirectives is the [Directives] companion. Tracks per-method
// state so DelegateTo tests can assert calls flow through to the
// inner impl.
//
// Several methods carry directives whose contracts conflict with the
// shape baseline applied by the suite generator (Submit/Wrap declare
// //testkit:errors so RejectInvalid expects sentinels on zero-valued
// inputs; Retry declares retry-succeeds-on-attempt requiring
// N-1 transient failures yet Writer's WriteSucceeds expects success
// on the first call). The in-mem honors both by branching on
// invalid-input shape and by exposing a retryFailMode flag that the
// suite's WithRetryFactory option enables for the retry contract
// only.
type InMemoryDirectives struct {
	mu sync.Mutex

	opened bool
	closed bool

	items map[string]Record

	// retryAttempt tracks the attempt counter for [Retry] so tests
	// can assert the stub's retry-succeeds sequencing matches the
	// directive's third-call success contract.
	retryAttempt map[string]int

	// retryFailMode flips Retry to its transient-failure semantics.
	// When false (default), Retry succeeds on every call so the
	// Writer baseline's WriteSucceeds/Idempotent passes. When true
	// (set on the WithRetryFactory impl), Retry returns ErrInternal
	// on the first N-1 calls and succeeds on the Nth.
	retryFailMode bool
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

// SeedAt seeds items[key] = item — keying by the explicit param. The
// suite's e2e companion uses this to satisfy Read's Reader baseline
// (which calls with sample key "test-key" and expects sample value).
func (d *InMemoryDirectives) SeedAt(key string, item Record) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items[key] = item
}

// SetRetryFailMode flips Retry into transient-failure mode. The suite
// e2e companion enables this on the WithRetryFactory impl so
// AssertRetrySucceedsOnAttempt observes N-1 failures then success.
func (d *InMemoryDirectives) SetRetryFailMode(fail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.retryFailMode = fail
}

// Opened reports whether Open has been called — used by stub
// auto-tests to verify order-after's prerequisite check.
func (d *InMemoryDirectives) Opened() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opened
}

// Open implements [Directives].
func (d *InMemoryDirectives) Open(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.opened = true
	return nil
}

// Close implements [Directives]. (integration-only — but the
// companion still maintains real state so tests that bypass the
// stub can verify behavior.)
func (d *InMemoryDirectives) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

// Submit implements [Directives]. Idempotent on repeated identical
// writes (Writer baseline passes). Returns errors.Join(ErrInternal,
// ErrConflict) when item.ID is empty so the //testkit:wrapped-via
// AND //testkit:errors contracts both observe their sentinels via
// errors.Is.
func (d *InMemoryDirectives) Submit(ctx context.Context, item Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item.ID == "" {
		return errors.Join(ErrInternal, ErrConflict)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items[item.ID] = item
	return nil
}

// Wrap implements [Directives]. Returns nil on valid input so the
// Writer baseline's WriteSucceeds passes; returns
// errors.Join(ErrInternal, ErrForbidden) on empty key so
// RejectInvalid + WrappedVia both fire.
func (d *InMemoryDirectives) Wrap(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if key == "" {
		return errors.Join(ErrInternal, ErrForbidden)
	}
	return nil
}

// Legacy implements [Directives]. (deprecated — but the companion
// still works; the stub's Deprecated annotation is purely
// documentation.)
func (d *InMemoryDirectives) Legacy(ctx context.Context, item Record) error {
	return d.Submit(ctx, item)
}

// Retry implements [Directives]. Default behavior (retryFailMode=
// false) succeeds on every call so Writer baseline contracts pass.
// Under retryFailMode (set by the suite's WithRetryFactory impl),
// returns ErrInternal on the first 2 calls and succeeds on the 3rd —
// matching //testkit:retry-succeeds-on-attempt 3.
func (d *InMemoryDirectives) Retry(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.retryFailMode {
		return nil
	}
	d.retryAttempt[key]++
	if d.retryAttempt[key] < 3 {
		return ErrInternal
	}
	return nil
}

// Read implements [Directives]. Honors //testkit:order-after Open by
// returning ErrNotFound when called before Open has run; that
// satisfies AssertOrderAfter's "fails before prerequisite" subtest.
// Once Open has been called, returns ErrNotFound only on real misses.
// The Reader baseline is suppressed for order-after methods (see
// the suite generator's per-method dispatcher) so the seeded factory
// state doesn't conflict with the pre-Open failure requirement.
func (d *InMemoryDirectives) Read(ctx context.Context, key string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.opened {
		return Record{}, ErrNotFound
	}
	v, ok := d.items[key]
	if !ok {
		return Record{}, ErrNotFound
	}
	return v, nil
}

// Shard implements [Directives].
func (d *InMemoryDirectives) Shard(ctx context.Context, item Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items[item.ID] = item
	return nil
}

// ShardByKey implements [Directives].
func (d *InMemoryDirectives) ShardByKey(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items[key] = Record{ID: key}
	return nil
}
