// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

//go:generate testkit stub -o allshapestest/allshapes.gen.go AllShapes
//go:generate testkit suite -o allshapestest/allshapes_spec.gen_test.go AllShapes
//go:generate testkit bench -o allshapestest/allshapes_bench.gen.go AllShapes

import (
	"context"
	"errors"
	"io"
	"iter"
)

// Record is the canonical value type across these fixtures. Plain
// string fields keep companions tiny while letting the generators
// exercise non-trivial typed parameters and returns.
type Record struct {
	ID    string
	Value string
}

// ErrNotFound is the sentinel reader-miss error. Methods declaring
// //testkit:errors ErrNotFound emit a FaultErrNotFound helper.
var ErrNotFound = errors.New("interfaces: not found")

// AllShapes carries one method per signature-tier shape from
// generator/shape. The numeric tags in each method's doc reference
// the priority entry in shape/registry.go's defaultDetectors —
// reviewers can cross-check the catalog against this fixture in one
// pass.
//
// The interface is intentionally large. It IS the comprehensive
// shape coverage; smaller per-shape fixtures fragment the surface
// without buying isolation the classifier doesn't already give.
type AllShapes interface {
	// 1000 — StreamReader (iter.Seq).
	All(ctx context.Context) iter.Seq[Record]

	// 1000 — StreamReader (iter.Seq2 with error).
	Scan(ctx context.Context) iter.Seq2[Record, error]

	// 950 — BatchReader (variadic key).
	Many(ctx context.Context, keys ...string) ([]Record, error)

	// 900 — StreamConsumer (interface-typed non-ctx param).
	ReadFrom(ctx context.Context, r io.Reader) (int, error)

	// 850 — Lookup (3 results, last bool).
	Inspect(ctx context.Context, key string) (Record, string, bool)

	// 840 — ReaderWithBool (2 results, last bool).
	Load(ctx context.Context, key string) (Record, bool)

	// 830 — PoisonAccessor (() error).
	Err() error

	// 820 — Predicate (() bool).
	IsHealthy() bool

	// 810 — VoidLifecycle (() void) +
	// //testkit:lifecycle-after-close (Reset is the close — paired
	// with Get, the post-Reset read returns the closed sentinel).
	//
	//testkit:lifecycle-after-close Get
	Reset()

	// 800 — Pure (() T).
	Description() string

	// 750 — MultiArgWriter (ctx + 3+ non-ctx + error).
	Schedule(ctx context.Context, key string, value Record, priority int) error

	// 700 — CompositeWriter (ctx, K, V) error +
	// //testkit:read-after-write (paired with Get — after Set(k, v),
	// Get(k) returns v).
	//
	//testkit:read-after-write Get
	Set(ctx context.Context, key string, value Record) error

	// 650 — MultiReader (ctx, K) (V1, V2, error).
	Fetch(ctx context.Context, key string) (Record, string, error)

	// 600 — MultiAggregator (ctx) (V1, V2, error).
	Stats(ctx context.Context) (int, int, error)

	// 550 — Deleter (ctx, K) error + //testkit:deleter +
	// //testkit:delete-removes (cross-method invariant).
	//
	//testkit:deleter
	//testkit:delete-removes Get
	Remove(ctx context.Context, key string) error

	// 500 — Writer (ctx, V) error +
	// //testkit:stream-reflects-mutations (paired with All — after
	// Put(item), the All() stream yields item).
	//
	//testkit:stream-reflects-mutations All
	Put(ctx context.Context, item Record) error

	// 450 — PointerReader (ctx, K) *V.
	Find(ctx context.Context, key string) *Record

	// 420 — Reader (ctx, K) (V, error).
	//
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key string) (Record, error)

	// 400 — ReaderNoError (ctx, K) V.
	Lookup(ctx context.Context, key string) Record

	// 350 — Aggregator (ctx) (T, error).
	Count(ctx context.Context) (int, error)

	// 300 — Mutator (ctx, V) void.
	Touch(ctx context.Context, key string)

	// 200 — Lifecycle (ctx) error.
	Init(ctx context.Context) error

	// — Unknown (catch-all): 3+ non-error returns exceeds
	// MultiAggregator's 2-result cap and falls through.
	Statistics(ctx context.Context) (int, int, int, error)
}
