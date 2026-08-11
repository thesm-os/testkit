// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/equivalence"
	"go.thesmos.sh/testkit/core/lawid"
)

// Cacheable verifies that repeated calls with the same key on the
// SUT return the same value within a single iteration. Caching may
// be implicit (memoized inside the impl) or explicit; either way
// the second call must agree with the first.
//
// AUTO-CACHEABLE fires for Readers carrying //testkit:cacheable.
type Cacheable[T any, K comparable, V any] struct {
	Read func(*rapid.T, T, K) (V, error)
	Keys *rapid.Generator[K]
}

// ID returns the stable identifier for this law.
func (Cacheable[T, K, V]) ID() string { return lawid.Cacheable }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (Cacheable[T, K, V]) REQID() string { return "" }

// Check verifies the law holds for the given SUT and reference.
// The reference parameter is unused — Cacheable is a self-consistency
// property of the SUT.
func (l Cacheable[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "Cacheable_key")
	v1, err1 := l.Read(rt, sut, k)
	v2, err2 := l.Read(rt, sut, k)
	if (err1 == nil) != (err2 == nil) {
		return fmt.Errorf("cacheable law: key %v: first err=%v, second err=%v", k, err1, err2)
	}
	if err1 != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if diff := cmp.Diff(v1, v2); diff != "" {
		return fmt.Errorf("cacheable law: key %v: repeated call disagrees (-first +second):\n%s", k, diff)
	}
	return nil
}

// DefaultOnError verifies that whenever the SUT's Read returns an
// error, the observed value is the configured default expression.
// Auto-emitted for Readers carrying //testkit:default-on-error.
type DefaultOnError[T any, K comparable, V any] struct {
	Read    func(*rapid.T, T, K) (V, error)
	Keys    *rapid.Generator[K]
	Default V

	// Eq is the equivalence the observed value is held to. Nil is strict
	// deep equality, which is the right default and the reason a generated
	// binding leaves this unset; supply a chain where the value carries a
	// field strict equality would wrongly reject, such as a timestamp.
	Eq *equivalence.Chain
}

// ID returns the stable identifier for this law.
func (DefaultOnError[T, K, V]) ID() string { return lawid.DefaultOnError }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (DefaultOnError[T, K, V]) REQID() string { return "" }

// Check verifies that error returns coincide with the default value.
func (l DefaultOnError[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "DefaultOnError_key")
	v, err := l.Read(rt, sut, k)
	if err == nil {
		return nil
	}
	if diff := l.Eq.Diff(l.Default, v); diff != "" {
		return fmt.Errorf("DefaultOnError: key %v: err=%v but the value is not the default (-default +got):\n%s",
			k, err, diff)
	}
	return nil
}

// PointInTime verifies snapshot semantics: a read taken at time t
// returns the value committed at or before t, regardless of writes
// happening at t' > t in concurrent goroutines.
//
// Encoded as a self-consistency check: two reads of the same key
// in immediate succession return the same value even when a third
// goroutine writes between them (the writer goroutine is provided
// by the consumer via Disturb; the law itself only verifies
// stability across the pair).
type PointInTime[T any, K comparable, V any] struct {
	Read    func(*rapid.T, T, K) (V, error)
	Keys    *rapid.Generator[K]
	Disturb func(*rapid.T, T, K) // optional concurrent disturber
}

// ID returns the stable identifier for this law.
func (PointInTime[T, K, V]) ID() string { return lawid.PointInTime }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (PointInTime[T, K, V]) REQID() string { return "" }

// Check verifies stability of consecutive reads under optional
// concurrent disturbance.
func (l PointInTime[T, K, V]) Check(rt *rapid.T, sut, ref T) error {
	k := l.Keys.Draw(rt, "PointInTime_key")
	v1, err1 := l.Read(rt, sut, k)
	if l.Disturb != nil {
		// The disturbance lands on both sides — the mirrored half of the
		// [Law] conduct contract. Disturb reports nothing, so there is no
		// refusal to relay; a divergence it causes is the next action's to
		// find, on a pair that at least saw the same calls.
		l.Disturb(rt, sut, k)
		l.Disturb(rt, ref, k)
	}
	v2, err2 := l.Read(rt, sut, k)
	if (err1 == nil) != (err2 == nil) {
		return fmt.Errorf("PointInTime: key %v: first err=%v, second err=%v", k, err1, err2)
	}
	if err1 != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if diff := cmp.Diff(v1, v2); diff != "" {
		return fmt.Errorf("PointInTime: key %v: snapshot drifted (-first +second):\n%s", k, diff)
	}
	return nil
}

// Sticky verifies that the first observed value for a key persists
// across subsequent reads — once a Reader has resolved a key, it
// keeps returning the same value. Auto-emitted for Readers
// carrying //testkit:sticky <Key>.
//
// The law caches the first non-error result per key across Check
// invocations. Sticky is a [StatefulLaw] in spirit but accepts the
// step argument implicitly via internal state.
type Sticky[T any, K comparable, V any] struct {
	Read func(*rapid.T, T, K) (V, error)
	Keys *rapid.Generator[K]

	// Eq is the equivalence successive reads are held to. Nil is strict
	// deep equality; supply a chain where a value legitimately carries a
	// field that moves between reads.
	Eq *equivalence.Chain

	first map[K]V // populated by Check
}

// ID returns the stable identifier for this law.
func (*Sticky[T, K, V]) ID() string { return lawid.Sticky }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*Sticky[T, K, V]) REQID() string { return "" }

// Check verifies the first-resolved value for k persists.
func (l *Sticky[T, K, V]) Check(rt *rapid.T, sut, _ T) error {
	k := l.Keys.Draw(rt, "Sticky_key")
	v, err := l.Read(rt, sut, k)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if l.first == nil {
		l.first = make(map[K]V)
	}
	if prior, ok := l.first[k]; ok {
		if diff := l.Eq.Diff(prior, v); diff != "" {
			return fmt.Errorf("sticky law: key %v: the resolved value changed (-first +now):\n%s", k, diff)
		}
		return nil
	}
	l.first[k] = v
	return nil
}

// MonotonicNonDecreasing verifies that an Aggregator-class method's
// result never decreases across calls. Auto-emitted for
// Aggregator/Appender methods carrying //testkit:monotonic.
//
// The law remembers the previous observation in self-state; the
// first call has nothing to compare against and trivially passes.
type MonotonicNonDecreasing[T any, R any] struct {
	Read func(*rapid.T, T) (R, error)
	Less func(a, b R) bool // returns true when a < b

	prev R
	seen bool
}

// ID returns the stable identifier for this law.
func (*MonotonicNonDecreasing[T, R]) ID() string { return lawid.MonotonicNonDecreasing }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (*MonotonicNonDecreasing[T, R]) REQID() string { return "" }

// Check verifies the result has not decreased.
func (l *MonotonicNonDecreasing[T, R]) Check(rt *rapid.T, sut, _ T) error {
	cur, err := l.Read(rt, sut)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if l.seen && l.Less(cur, l.prev) {
		return fmt.Errorf("MonotonicNonDecreasing: previous=%v, now=%v", l.prev, cur)
	}
	l.prev = cur
	l.seen = true
	return nil
}

// Sentinel sentinels for use by Reader-class laws that need to
// distinguish "absent" from "errored" without a pre-injected
// not-found error.
var (
	// ErrSentinelAbsent is used as a generic "not present" marker
	// by laws that compose absence semantics. Consumers may
	// configure their own; this is the default.
	ErrSentinelAbsent = errors.New("law: sentinel absent")
)
