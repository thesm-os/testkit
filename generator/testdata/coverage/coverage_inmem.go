// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage

import (
	"context"
	"sync"

	"go.thesmos.sh/testkit/suite"
)

// scopeKey is the context key used by the scope-directive contract
// to thread the granted scope through Privileged. The companion
// test's WithScopeContext closure stores the scope under this key;
// Privileged reads it.
type scopeKey struct{}

// inmem is the [Coverage] companion. Built to satisfy both the
// shape baselines (returns sample literals on first call) and the
// directive contracts (hooks fire, leases acquire/release, etc.).
type inmem struct {
	mu sync.Mutex

	items map[string]Item

	// description is what Description returns. Default
	// "test-result0" matches the suite's Pure-baseline sample.
	description string

	// version is what Version returns. Default 42 matches the
	// Aggregator-baseline sample; monotonic non-decreasing is
	// trivially satisfied by a constant.
	version int

	// counter is what Counter returns. Default 42; concurrent fanout
	// reads it under the mutex.
	counter int

	// quick is what Quick returns. Default 42.
	quick int

	// convergent is what Convergent returns. Default 42 — two
	// consecutive calls return the same value, satisfying
	// AssertEventuallyConverges immediately.
	convergent int

	// auditedCount is the side-effect counter Audited returns. Starts
	// at 42 (Aggregator-baseline sample); Track() increments so
	// AssertSideEffectObservable's before/after differ.
	auditedCount int

	// leaseHeld is the lease state. true after AcquireLease, flipped
	// false on ReleaseLease.
	leaseHeld bool
}

// NewInMem returns an in-mem with default state aligned to the
// suite contracts: items pre-seeded under "test-key", aggregate
// returns at 42, lease released. Each fresh impl from this
// factory passes both the shape baselines and the directive
// contracts under the companion test's option configuration.
func NewInMem() *inmem {
	return &inmem{
		items:        map[string]Item{"test-key": {ID: "test-id"}},
		description:  "test-result0",
		version:      42,
		counter:      42,
		quick:        42,
		convergent:   42,
		auditedCount: 42,
	}
}

// Description implements [Coverage]. Returns a stable string so
// Pure's deterministic + impl-independent contracts pass.
func (m *inmem) Description() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.description
}

// GetCached implements [Coverage]. Reads from the items map; under
// //testkit:cacheable the runtime asserts three sequential reads
// return equal values, which holds because the map is stable
// between calls.
func (m *inmem) GetCached(ctx context.Context, key string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[key]
	if !ok {
		return Item{}, ErrInvalid
	}
	return v, nil
}

// Version implements [Coverage]. Returns the monotonic version —
// constant 42 is non-decreasing and lies in [0..1000].
func (m *inmem) Version(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.version, nil
}

// Counter implements [Coverage]. Returns a constant; concurrent-
// safe via the mutex.
func (m *inmem) Counter(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counter, nil
}

// Snapshot implements [Coverage]. ReaderNoError contract; concurrent-
// readers contract reads in parallel.
func (m *inmem) Snapshot(ctx context.Context, key string) Item {
	if ctx.Err() != nil {
		return Item{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[key]
}

// Push implements [Coverage]. Nilsafe: nil-pointer input no-ops
// without panic.
func (m *inmem) Push(ctx context.Context, item *Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = *item
	return nil
}

// Validate implements [Coverage]. Empty ID returns ErrInvalid for
// the //testkit:validates contract; populated IDs succeed.
func (m *inmem) Validate(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item.ID == "" {
		return ErrInvalid
	}
	return nil
}

// Quick implements [Coverage]. Returns immediately so the
// //testkit:timeout 100ms contract passes.
func (m *inmem) Quick(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.quick, nil
}

// Track implements [Coverage]. Increments auditedCount so the
// //testkit:side-effect contract observes a state change through
// Audited.
func (m *inmem) Track(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	m.auditedCount++
	return nil
}

// Audited implements [Coverage]. Reads the side-effect counter.
// Starts at 42 (aligned to the Aggregator-baseline sample); each
// Track() increments it so before/after observation differs.
func (m *inmem) Audited(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.auditedCount, nil
}

// AcquireLease implements [Coverage]. Returns nil on first acquire,
// ErrLeaseHeld when already held — the lease contract's
// double-acquire-without-release case.
func (m *inmem) AcquireLease(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.leaseHeld {
		return ErrLeaseHeld
	}
	m.leaseHeld = true
	return nil
}

// ReleaseLease implements [Coverage]. Releases the lease so the
// next AcquireLease succeeds.
func (m *inmem) ReleaseLease(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaseHeld = false
	return nil
}

// Publish implements [Coverage]. Reads the [suite.HookRecorder] from
// ctx (test-side wiring) and records the AfterPublish hook firing.
// Production callers that don't supply a recorder see a no-op via
// the recorder's nil guard.
func (m *inmem) Publish(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	suite.RecorderFromContext(ctx).Record("AfterPublish")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	return nil
}

// Convergent implements [Coverage]. Returns a constant so two
// consecutive polled samples are equal and AssertEventuallyConverges
// terminates on the first comparison.
func (m *inmem) Convergent(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.convergent, nil
}

// Privileged implements [Coverage]. Reads the granted scope from
// ctx via [scopeKey]; returns ErrUnauthorized when the scope key is
// absent, nil otherwise. The companion test's WithScopeContext
// stores "admin" under scopeKey.
func (m *inmem) Privileged(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v := ctx.Value(scopeKey{}); v == nil {
		return ErrUnauthorized
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	return nil
}

// ScopeContext returns a context-decorator function suitable for
// [suite.WithScopeContext]. Companion tests pass this so the scope
// directive contract has a typed builder.
func ScopeContext(parent context.Context) func(scope string) context.Context {
	return func(scope string) context.Context {
		return context.WithValue(parent, scopeKey{}, scope)
	}
}
