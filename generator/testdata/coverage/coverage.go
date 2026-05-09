// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package coverage is the testdata fixture exercising directives
// that lack a fit on the AllShapes / Directives fixtures (whose
// methods are already saturated with shape baselines or the small
// directive set those fixtures were originally designed for). Each
// method here picks one or two directives so the suite generator
// emits the corresponding AssertX subtest against a real in-mem
// implementation.
//
// Directives covered:
//   - pure / cacheable / monotonic on read-shape methods
//   - concurrent / concurrent-readers / nilsafe markers
//   - validates / bounded / timeout (signature- or option-driven)
//   - sideeffect / lease (paired-method invariants)
//   - hooks / scope / eventually (option-driven)
//
// Pagination is deferred — its runtime helper expects a Page struct
// shape (Items + Cursor) that doesn't fit naturally on the iter.Seq
// interfaces this package favors. A standalone pagination fixture
// will land separately.
package coverage

//go:generate testkit suite -o coveragetest/coverage_spec.gen_test.go Coverage

import (
	"context"
	"errors"
)

// Sentinels used by the directive contracts.
var (
	ErrInvalid      = errors.New("coverage: invalid")
	ErrUnauthorized = errors.New("coverage: unauthorized")
	ErrLeaseHeld    = errors.New("coverage: lease already held")
)

// Item is the value type for the lease + side-effect contracts.
type Item struct {
	ID    string
	Value string
}

// Coverage carries one method per directive that needed e2e
// coverage. Each method's directive set is the minimum needed to
// trigger the corresponding suite.AssertX subtest; in-mem behavior
// in coverage_inmem.go realizes the contract.
type Coverage interface {
	// Pure-shape, //testkit:pure — impl-independent.
	//
	//testkit:pure
	Description() string

	// Reader-shape, //testkit:cacheable — repeated reads return
	// equal values.
	//
	//testkit:cacheable
	GetCached(ctx context.Context, key string) (Item, error)

	// Aggregator-shape, //testkit:monotonic — result never decreases.
	// Combined with //testkit:bounded so the AggregatorBounds
	// option-gated extra fires.
	//
	//testkit:monotonic
	//testkit:bounded 0..1000
	Version(ctx context.Context) (int, error)

	// Aggregator-shape, //testkit:concurrent — 16×25 strict fanout.
	//
	//testkit:concurrent
	Counter(ctx context.Context) (int, error)

	// ReaderNoError-shape, //testkit:concurrent-readers — 32 readers.
	//
	//testkit:concurrent-readers
	Snapshot(ctx context.Context, key string) Item

	// Writer-shape, //testkit:nilsafe — handles nil-bearing inputs
	// without panic.
	//
	//testkit:nilsafe
	Push(ctx context.Context, item *Item) error

	// Writer-shape, //testkit:validates ID — empty ID returns error.
	//
	//testkit:errors ErrInvalid
	//testkit:validates ID
	Validate(ctx context.Context, item Item) error

	// Aggregator-shape, //testkit:timeout — completes within
	// declared deadline.
	//
	//testkit:timeout 100ms
	Quick(ctx context.Context) (int, error)

	// Writer-shape, //testkit:side-effect Method=Audited — Track
	// observability through Audited.
	//
	//testkit:side-effect Audited
	Track(ctx context.Context, item Item) error

	// Aggregator-shape — counterpart to Track, observes side-effects.
	Audited(ctx context.Context) (int, error)

	// Lifecycle-shape, //testkit:lease Release=ReleaseLease —
	// acquire returns nil; ReleaseLease releases.
	//
	//testkit:errors ErrLeaseHeld
	//testkit:lease ReleaseLease
	AcquireLease(ctx context.Context) error

	// Lifecycle-shape — paired release for AcquireLease.
	ReleaseLease(ctx context.Context) error

	// Writer-shape, //testkit:hooks AfterPublish — fires the named
	// hook(s) during execution. The contract drives the hook
	// recorder option.
	//
	//testkit:hooks AfterPublish
	Publish(ctx context.Context, item Item) error

	// Aggregator-shape, //testkit:eventually 200ms — converges to a
	// stable value within the declared deadline.
	//
	//testkit:eventually 200ms
	Convergent(ctx context.Context) (int, error)

	// Writer-shape, //testkit:scope admin — calls without the named
	// scope return ErrUnauthorized.
	//
	//testkit:errors ErrUnauthorized
	//testkit:scope admin
	Privileged(ctx context.Context, item Item) error
}
