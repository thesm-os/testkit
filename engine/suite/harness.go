// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"errors"
	"fmt"
	"testing"
)

// The policy every generated harness shares, in one home. A harness's
// TYPED surface — which capability fields exist, what their constructors
// return — is irreducibly per-interface and stays in the template; the
// rules beneath it (exactly one constructor, exclusive clocked pair,
// distinct pools) were being re-emitted per interface, which is the drift
// class this library exists to kill.

// OneCtor enforces the constructor pair's contract — exactly one of the
// plain and lifecycle forms — and returns the unified build function.
func OneCtor[T any](name string, plain func() T, start func(tb testing.TB) T) (func(testing.TB) T, error) {
	switch {
	case name == "":
		return nil, errors.New("a harness has no Name")
	case plain == nil && start == nil:
		return nil, fmt.Errorf("harness %q sets neither New nor Start; set exactly one", name)
	case plain != nil && start != nil:
		return nil, fmt.Errorf("harness %q sets both New and Start; set exactly one", name)
	case start != nil:
		return start, nil
	default:
		return func(testing.TB) T { return plain() }, nil
	}
}

// ExclusivePair refuses a harness that set both members of an
// either-or field pair, naming them.
func ExclusivePair(name, fieldA, fieldB string, aSet, bSet bool) error {
	if aSet && bSet {
		return fmt.Errorf("harness %q sets both %s and %s; set at most one", name, fieldA, fieldB)
	}
	return nil
}

// Inductions maps sentinels to the triggers that put a subject into the
// state each sentinel names. The trigger receives the sentinel it is
// registered under, so a method taking the error is a map value — no
// closure, no restated sentinel. Generated packages alias this under
// their own name, keeping consumer vocabulary local while the type has
// one home.
type Inductions[T any] map[error]func(s T, sentinel error)

// LowerInductions lowers a harness's typed induction map onto the
// runtime subject's shape: the concrete type is recovered by assertion,
// because the harness speaks T and the runner speaks the interface.
// The assertion cannot fail for a subject built by its own harness; the
// guard exists for the day a constructor and a trigger disagree.
func LowerInductions[S, T any](name string, in Inductions[T]) map[error]func(testing.TB, S) {
	if len(in) == 0 {
		return nil
	}
	out := make(map[error]func(testing.TB, S), len(in))
	for sentinel, trigger := range in {
		out[sentinel] = func(tb testing.TB, s S) {
			concrete, ok := downcast[S, T](s)
			if !ok {
				tb.Fatalf("internal: subject %q is not the type its constructor returns"+
					" (or an Unwrap() chain to it)", name)
				return
			}
			trigger(concrete, sentinel)
		}
	}
	return out
}

// LowerRecover adapts a harness's typed Recover to the subject's
// interface-typed one — the same downcast [LowerInductions] makes, for
// the same reason: the harness speaks the constructor's concrete type,
// the subject speaks the interface. A nil recover lowers to nil, so
// the caller assigns unconditionally.
func LowerRecover[S, T any](name string, reopen func(testing.TB, T) T) func(testing.TB, S) S {
	if reopen == nil {
		return nil
	}
	return func(tb testing.TB, prior S) S {
		concrete, ok := downcast[S, T](prior)
		if !ok {
			tb.Fatalf("internal: subject %q recovered over an instance of another type"+
				" (or an Unwrap() chain to it)", name)
			var zero S
			return zero
		}
		return any(reopen(tb, concrete)).(S)
	}
}

// ExcuseSet lowers a harness's excuse list onto the subject's map, nil
// for empty so an unexcused subject carries no allocation.
func ExcuseSet(ids []ID) map[ID]bool {
	if len(ids) == 0 {
		return nil
	}
	out := make(map[ID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

// Must is the panic wrapper for config paths that already validated in
// the option loop, or that have no error channel (Suite as data). The
// panic is the invariant's last resort, not the reporting surface —
// misuse is reported where the option arrived.
func Must[C any](c C, err error) C {
	if err != nil {
		// The invariant's last resort: every caller validated in the
		// option loop, so reaching this is a bug in the emitted
		// defaults, not a runtime condition.
		panic(err.Error()) //nolint:forbidigo // see above
	}
	return c
}

// DistinctPool applies the drawn-pool policy: a nil pool takes the
// derived default, and a supplied pool must hold at least two DISTINCT
// values — length alone let a pool that repeats one value make every
// comparison that needs two vacuous.
func DistinctPool[V comparable](field string, pool, derived []V) ([]V, error) {
	if pool == nil {
		pool = derived
	}
	seen := map[V]bool{}
	for _, v := range pool {
		seen[v] = true
		if len(seen) >= 2 {
			return pool, nil
		}
	}
	return nil, fmt.Errorf(
		"%s needs at least two distinct values; a pool that repeats one value "+
			"makes every comparison that depends on two vacuous", field)
}

// downcast resolves s to the harness's concrete type, following
// Unwrap() chains so a decorator-wrapped subject — tracing, metrics, a
// consumer's logging shim — still reaches the instance its trigger or
// recover was written against. The chain is bounded to keep a cyclic
// Unwrap from hanging a test.
func downcast[S, T any](s S) (T, bool) {
	cur := any(s)
	for range 32 {
		if concrete, ok := cur.(T); ok {
			return concrete, true
		}
		u, ok := cur.(interface{ Unwrap() S })
		if !ok {
			break
		}
		cur = any(u.Unwrap())
	}
	var zero T
	return zero, false
}
