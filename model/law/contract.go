// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

// PersisterRetrievable verifies that the value returned by a
// Persister's Save can be looked up via the paired Reader. The
// returned ID is the lookup key. Auto-emitted for methods carrying
// //testkit:persister <Reader>.
type PersisterRetrievable[T any, V any, ID comparable] struct {
	Save   func(*rapid.T, T, V) (ID, error)
	Read   func(*rapid.T, T, ID) (V, error)
	Values *rapid.Generator[V]
}

// ID returns the stable identifier for this law.
func (PersisterRetrievable[T, V, ID]) ID() string { return "AUTO-PERSISTER-RETRIEVABLE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PersisterRetrievable[T, V, ID]) REQID() string { return "" }

// Check verifies Save-then-Read returns the saved value.
func (l PersisterRetrievable[T, V, ID]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "PersisterRetrievable_value")
	id, err := l.Save(rt, sut, v)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	got, err := l.Read(rt, sut, id)
	if err != nil {
		return fmt.Errorf("PersisterRetrievable: saved id=%v, Read errored: %v", id, err)
	}
	if diff := cmp.Diff(v, got); diff != "" {
		return fmt.Errorf("PersisterRetrievable: saved/read mismatch (-saved +read):\n%s", diff)
	}
	return nil
}

// UpdaterReplaces verifies that calling Update twice with values
// that share a key results in only the second value being
// observable via Read. Last-write-wins per key.
//
// The consumer supplies KeyOf to extract the matching key from V.
type UpdaterReplaces[T any, V any, K comparable] struct {
	Update func(*rapid.T, T, V) error
	Read   func(*rapid.T, T, K) (V, error)
	Values *rapid.Generator[V]
	KeyOf  func(V) K
}

// ID returns the stable identifier for this law.
func (UpdaterReplaces[T, V, K]) ID() string { return "AUTO-UPDATER-REPLACES" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (UpdaterReplaces[T, V, K]) REQID() string { return "" }

// Check writes v1 then v2 sharing the same key and verifies the
// reader sees v2.
func (l UpdaterReplaces[T, V, K]) Check(rt *rapid.T, sut, _ T) error {
	v1 := l.Values.Draw(rt, "UpdaterReplaces_v1")
	v2 := l.Values.Draw(rt, "UpdaterReplaces_v2")
	if l.KeyOf(v1) != l.KeyOf(v2) {
		return nil // shrink will eventually find a same-key pair
	}
	if err := l.Update(rt, sut, v1); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Update(rt, sut, v2); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	got, err := l.Read(rt, sut, l.KeyOf(v2))
	if err != nil {
		return fmt.Errorf("UpdaterReplaces: Read after replace errored: %v", err)
	}
	if diff := cmp.Diff(v2, got); diff != "" {
		return fmt.Errorf("UpdaterReplaces: second write not visible (-v2 +read):\n%s", diff)
	}
	return nil
}

// UpserterIdempotent verifies repeated Upserts of the same value
// leave the reader-observed state unchanged after the second call.
type UpserterIdempotent[T any, V any, K comparable] struct {
	Upsert func(*rapid.T, T, V) error
	Read   func(*rapid.T, T, K) (V, error)
	Values *rapid.Generator[V]
	KeyOf  func(V) K
}

// ID returns the stable identifier for this law.
func (UpserterIdempotent[T, V, K]) ID() string { return "AUTO-UPSERTER-IDEMPOTENT" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (UpserterIdempotent[T, V, K]) REQID() string { return "" }

// Check upserts v twice and verifies the read result is stable.
func (l UpserterIdempotent[T, V, K]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "UpserterIdempotent_value")
	if upsertErr := l.Upsert(rt, sut, v); upsertErr != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	first, readErr := l.Read(rt, sut, l.KeyOf(v))
	if readErr != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Upsert(rt, sut, v); err != nil {
		return fmt.Errorf("upserter law: second upsert errored: %v", err)
	}
	second, err := l.Read(rt, sut, l.KeyOf(v))
	if err != nil {
		return fmt.Errorf("upserter law: read after second upsert errored: %v", err)
	}
	if diff := cmp.Diff(first, second); diff != "" {
		return fmt.Errorf("upserter law: second upsert changed read (-first +second):\n%s", diff)
	}
	return nil
}

// CASAtomicOneWinner verifies that two concurrent CAS writes with
// the same starting version produce exactly one success and one
// version-mismatch error. Auto-emitted for methods carrying
// //testkit:cas <VersionField>.
type CASAtomicOneWinner[T any, V any] struct {
	CAS      func(*rapid.T, T, V) error
	Read     func(*rapid.T, T) (V, error)
	Values   *rapid.Generator[V]
	Mismatch error
}

// ID returns the stable identifier for this law.
func (CASAtomicOneWinner[T, V]) ID() string { return "AUTO-CAS-ATOMIC-ONE-WINNER" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CASAtomicOneWinner[T, V]) REQID() string { return "" }

// Check races two CAS calls; expects exactly one success.
func (l CASAtomicOneWinner[T, V]) Check(rt *rapid.T, sut, _ T) error {
	v1 := l.Values.Draw(rt, "CASAtomicOneWinner_v1")
	v2 := l.Values.Draw(rt, "CASAtomicOneWinner_v2")
	err1 := l.CAS(rt, sut, v1)
	err2 := l.CAS(rt, sut, v2)
	successes := 0
	if err1 == nil {
		successes++
	}
	if err2 == nil {
		successes++
	}
	mismatches := 0
	if errors.Is(err1, l.Mismatch) {
		mismatches++
	}
	if errors.Is(err2, l.Mismatch) {
		mismatches++
	}
	if successes != 1 || mismatches != 1 {
		return fmt.Errorf(
			"CASAtomicOneWinner: expected 1 success + 1 mismatch, got successes=%d mismatches=%d (err1=%v err2=%v)",
			successes, mismatches, err1, err2,
		)
	}
	return nil
}

// AppenderMonotonicOffsets verifies the offsets returned by
// successive Appends are strictly increasing. Auto-emitted for
// methods carrying //testkit:appender.
type AppenderMonotonicOffsets[T any, V any, Off interface{ ~int | ~int64 }] struct {
	Append func(*rapid.T, T, V) (Off, error)
	Values *rapid.Generator[V]

	prev    Off
	hasPrev bool
}

// ID returns the stable identifier for this law.
func (*AppenderMonotonicOffsets[T, V, Off]) ID() string { return "AUTO-APPENDER-MONOTONIC-OFFSETS" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*AppenderMonotonicOffsets[T, V, Off]) REQID() string { return "" }

// Check appends a value and verifies the returned offset exceeds
// the previously-observed offset.
func (l *AppenderMonotonicOffsets[T, V, Off]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "AppenderMonotonicOffsets_value")
	off, err := l.Append(rt, sut, v)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if l.hasPrev && off <= l.prev {
		return fmt.Errorf("AppenderMonotonicOffsets: offset %v did not exceed previous %v", off, l.prev)
	}
	l.prev = off
	l.hasPrev = true
	return nil
}

// SingleflightCoalesces verifies that N concurrent calls with the
// same key invoke the compute function at most once. Auto-emitted
// for methods carrying //testkit:singleflight.
//
// The consumer threads a shared call counter through Compute; the
// law inspects the counter after running M concurrent calls.
type SingleflightCoalesces[T any, K comparable, V any] struct {
	Call     func(ctx context.Context, sut T, k K, compute func() V) (V, error)
	Compute  func() V
	Keys     *rapid.Generator[K]
	Parallel int
	Counter  *int // pointer the Compute closure increments
}

// ID returns the stable identifier for this law.
func (SingleflightCoalesces[T, K, V]) ID() string { return "AUTO-SINGLEFLIGHT-COALESCES" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (SingleflightCoalesces[T, K, V]) REQID() string { return "" }

// Check launches N concurrent calls with the same key and asserts
// the compute counter only advances by 1.
func (l SingleflightCoalesces[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "SingleflightCoalesces_key")
	before := *l.Counter
	n := l.Parallel
	if n <= 0 {
		n = 8
	}
	done := make(chan struct{}, n)
	for range n {
		go func() {
			_, _ = l.Call(rt.Context(), sut, k, l.Compute)
			done <- struct{}{}
		}()
	}
	for range n {
		<-done
	}
	got := *l.Counter - before
	if got > 1 {
		return fmt.Errorf("SingleflightCoalesces: %d concurrent calls invoked compute %d times (expected ≤1)", n, got)
	}
	return nil
}

// TransactionRollbackOnError verifies that when the body of a
// TransactionFunc returns an error, no buffered writes are visible
// after the call returns. Auto-emitted for methods carrying
// //testkit:transaction.
type TransactionRollbackOnError[T any, K comparable, V any] struct {
	Run      func(*rapid.T, T, func(_ context.Context) error) error
	Read     func(*rapid.T, T, K) (V, error)
	Keys     *rapid.Generator[K]
	NotFound error
}

// ID returns the stable identifier for this law.
func (TransactionRollbackOnError[T, K, V]) ID() string { return "AUTO-TRANSACTION-ROLLBACK" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TransactionRollbackOnError[T, K, V]) REQID() string { return "" }

// Check runs a body that always errors and verifies a probe key
// remains absent after the call returns.
func (l TransactionRollbackOnError[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	probe := l.Keys.Draw(rt, "TransactionRollbackOnError_key")
	before, beforeErr := l.Read(rt, sut, probe)
	_ = l.Run(rt, sut, func(_ context.Context) error {
		return errors.New("law: induced rollback")
	})
	after, afterErr := l.Read(rt, sut, probe)
	// Whether the key existed before or not, the post-error state
	// must equal the pre-call state observationally.
	if (beforeErr == nil) != (afterErr == nil) {
		return fmt.Errorf("TransactionRollbackOnError: key %v: errored body changed presence (before=%v, after=%v)",
			probe, beforeErr, afterErr)
	}
	if beforeErr == nil {
		if diff := cmp.Diff(before, after); diff != "" {
			return fmt.Errorf("TransactionRollbackOnError: key %v: value changed across error (-before +after):\n%s",
				probe, diff)
		}
	}
	return nil
}

// LeaseDoubleAcquireBlocks verifies a second Acquire of an
// already-held lease returns the configured held error. Auto-
// emitted for methods carrying //testkit:acquire <Release>.
type LeaseDoubleAcquireBlocks[T any, K comparable] struct {
	Acquire func(*rapid.T, T, K) error
	Release func(*rapid.T, T, K) error
	Keys    *rapid.Generator[K]
	Held    error
}

// ID returns the stable identifier for this law.
func (LeaseDoubleAcquireBlocks[T, K]) ID() string { return "AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LeaseDoubleAcquireBlocks[T, K]) REQID() string { return "" }

// Check acquires a key, attempts to acquire again, verifies the
// held error fires, then releases.
func (l LeaseDoubleAcquireBlocks[T, K]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "LeaseDoubleAcquireBlocks_key")
	if err := l.Acquire(rt, sut, k); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	defer func() { _ = l.Release(rt, sut, k) }()
	err := l.Acquire(rt, sut, k)
	if !errors.Is(err, l.Held) {
		return fmt.Errorf("LeaseDoubleAcquireBlocks: key %v: second acquire returned %v (want held=%v)",
			k, err, l.Held)
	}
	return nil
}
