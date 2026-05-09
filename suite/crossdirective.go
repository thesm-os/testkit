// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"go.thesmos.sh/testkit"
)

// AssertReadAfterWriteByKey verifies that calling the writer with a
// (key, value) pair makes the named reader return that value. The
// caller supplies typed closures for both methods. Distinct from
// [AssertReadAfterWrite] (which uses an extractKey callback): this
// variant is for the //testkit:read-after-write directive on
// CompositeWriter-shape methods, where the writer takes (K, V)
// directly.
func AssertReadAfterWriteByKey[T any, K, V comparable](
	key K,
	value V,
	write func(ctx context.Context, impl T, k K, v V) error,
	read func(ctx context.Context, impl T, k K) (V, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("read-after-write", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			err := write(t.Context(), impl, key, value)
			testkit.NoError(t, err, "write must succeed")
			got, err := read(t.Context(), impl, key)
			testkit.NoError(t, err, "read after write must succeed")
			testkit.Equal(t, got, value, "read must return the written value")
		})
	}
}

// AssertDeleteRemovesByKey verifies that after the deleter runs,
// the named reader returns the configured not-found sentinel for
// the same key. Distinct from [AssertDeleteRemovesValue] (which
// extracts K from V): this variant takes the key directly,
// matching the //testkit:delete-removes directive's emission.
func AssertDeleteRemovesByKey[T any, K comparable, V any](
	key K,
	notFound error,
	del func(ctx context.Context, impl T, k K) error,
	read func(ctx context.Context, impl T, k K) (V, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("delete-removes", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			err := del(t.Context(), impl, key)
			testkit.NoError(t, err, "delete must succeed")
			_, err = read(t.Context(), impl, key)
			testkit.ErrorIs(t, err, notFound,
				"read after delete must surface the not-found sentinel")
		})
	}
}

// AssertStreamReflectsValueWritten verifies that after the writer
// runs with a single value, the named stream method yields that
// value. Distinct from [AssertStreamReflectsMutations] (which uses
// extractKey + del-and-recheck): this variant is the simpler
// directive-driven contract — write one value, scan, find it.
func AssertStreamReflectsValueWritten[T any, V comparable](
	value V,
	write func(ctx context.Context, impl T, v V) error,
	collect func(ctx context.Context, impl T) ([]V, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("stream-reflects-mutations", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			err := write(t.Context(), impl, value)
			testkit.NoError(t, err, "write must succeed")
			items, err := collect(t.Context(), impl)
			testkit.NoError(t, err, "stream collection must succeed")
			if !slices.Contains(items, value) {
				t.Fatalf("stream must yield the written value (got items: %v)", items)
			}
		})
	}
}

// AssertLifecycleAfterClose verifies that after the close method
// runs, the named reader returns either the configured closed
// sentinel or [context.Canceled] (whichever the impl uses).
func AssertLifecycleAfterClose[T any, K comparable, V any](
	key K,
	closed error,
	closeFn func(ctx context.Context, impl T) error,
	read func(ctx context.Context, impl T, k K) (V, error),
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("lifecycle-after-close", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			err := closeFn(t.Context(), impl)
			testkit.NoError(t, err, "close must succeed")
			_, err = read(t.Context(), impl, key)
			if err == nil || (!errors.Is(err, closed) && !errors.Is(err, context.Canceled)) {
				t.Fatalf("read after close must surface the closed sentinel or context.Canceled, got %v", err)
			}
		})
	}
}

// AssertLifecycleAfterCloseReflective is a reflection-driven variant
// of [AssertLifecycleAfterClose] for cases where the paired reader's
// signature isn't known at template emit time. Calls the reader by
// name via [reflect.Value.MethodByName] with [t.Context] alone;
// asserts the trailing error result is non-nil.
//
// Use this when the reader takes only ctx (Aggregator-shape). For
// readers that take args, use [AssertLifecycleAfterClose] with a
// typed closure.
func AssertLifecycleAfterCloseReflective[T any](
	readerName string,
	closeFn func(ctx context.Context, impl T) error,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("lifecycle-after-close ("+readerName+")", func(t *testing.T) {
			t.Parallel()
			impl := factory()
			testkit.NoError(t, closeFn(t.Context(), impl), "close must succeed")
			rv := reflect.ValueOf(impl).MethodByName(readerName)
			if !rv.IsValid() {
				t.Fatalf("lifecycle-after-close: reader %q not found on impl", readerName)
			}
			results := rv.Call([]reflect.Value{reflect.ValueOf(t.Context())})
			if len(results) == 0 {
				t.Fatalf("lifecycle-after-close: reader %q returned no values", readerName)
			}
			last := results[len(results)-1].Interface()
			err, ok := last.(error)
			if !ok || err == nil {
				t.Fatalf("reader after close must return a non-nil error (got %v)", last)
			}
		})
	}
}

// AssertCRDTMerge verifies that two impls applying the same set of
// merge operations in opposite orders converge to equal state. The
// caller supplies the merge closure; state equality falls back to
// [reflect.DeepEqual] when stateEqual is nil. CRDT merge laws
// (commutative, associative, idempotent) require the resulting
// states to be equal regardless of order.
func AssertCRDTMerge[T, V any](
	a, b V,
	merge func(ctx context.Context, impl T, v V) error,
	stateEqual func(impl1, impl2 T) bool,
) func(*testing.T, func() T) {
	return func(t *testing.T, factory func() T) {
		t.Helper()
		t.Run("crdt-merge (commutative)", func(t *testing.T) {
			t.Parallel()
			implA := factory()
			implB := factory()
			testkit.NoError(t, merge(t.Context(), implA, a), "implA: merge a must succeed")
			testkit.NoError(t, merge(t.Context(), implA, b), "implA: merge b must succeed")
			testkit.NoError(t, merge(t.Context(), implB, b), "implB: merge b must succeed")
			testkit.NoError(t, merge(t.Context(), implB, a), "implB: merge a must succeed")
			eq := stateEqual
			if eq == nil {
				eq = func(x, y T) bool { return reflect.DeepEqual(x, y) }
			}
			testkit.True(t, eq(implA, implB),
				"crdt-merge: opposite-order merges must converge to equal state")
		})
	}
}
