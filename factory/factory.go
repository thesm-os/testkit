// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package factory

import "fmt"

// Named pairs a constructor closure with a stable identifier. Used
// by every generator's `*AcrossImpls` entry point so failure
// messages cite the implementation by name rather than by positional
// index.
//
//	storetest.AssertStoreModelAcrossImpls(t,
//	    []storetest.StoreModelOption{...},
//	    factory.NewNamed("in-memory", newInMem),
//	    factory.NewNamed("redis",     newRedis),
//	)
//
// The runner constructs a fresh impl per iteration via the closure
// passed to NewNamed; the name flows into trace events,
// classified-failure JSON, and the coverage-summary header so
// consumers reading CI artifacts can tell which impl produced which
// observation without recovering the call site.
//
// Fields are unexported. Construction must go through [NewNamed],
// which validates that the name is non-empty and the factory is
// non-nil. Validating once at construction time prevents every
// runner from re-checking the same invariants downstream.
type Named[T any] struct {
	name    string
	factory func() T
}

// NewNamed constructs a [Named] from a name and a constructor
// closure. Panics when name is empty or fn is nil — both are usage
// errors the runner cannot recover from. The panic fires at test
// setup, before any iteration runs, so the diagnostic surfaces at
// the call site in test output rather than partway through a
// property run.
func NewNamed[T any](name string, fn func() T) Named[T] {
	if name == "" {
		//nolint:forbidigo // usage error: Named requires a non-empty identifier
		panic("factory.NewNamed: name is empty")
	}
	if fn == nil {
		//nolint:forbidigo // usage error: Named requires a non-nil constructor
		panic(fmt.Sprintf("factory.NewNamed: factory closure is nil (name=%q)", name))
	}
	return Named[T]{name: name, factory: fn}
}

// Name returns the identifier supplied to [NewNamed].
func (n Named[T]) Name() string {
	return n.name
}

// Construct calls the factory closure and returns a fresh value.
// Each call invokes the closure exactly once; the closure is
// expected to return isolated values (no shared state across calls)
// per the runner's per-iteration isolation guarantee.
func (n Named[T]) Construct() T {
	return n.factory()
}
