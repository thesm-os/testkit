// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

// defaultXSSTokens are literal tag-openers a correct HTML escaper
// converts (the leading '<' becomes "&lt;"), so their survival in
// rendered output signals unescaped, script-capable markup.
var defaultXSSTokens = []string{"<script", "<img", "<svg", "<iframe"}

// IdempotentWrite verifies that the second Write of an identical
// value is observably equivalent to the first — repeated writes
// of (key, value) produce the same state. Auto-emitted for
// Writers carrying //testkit:idempotent.
//
// The law performs the comparison via a consumer-supplied probe
// function that reads enough state to detect divergence (typically
// the paired Reader returning V for the same key).
type IdempotentWrite[T any, V any, Obs any] struct {
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (IdempotentWrite[T, V, Obs]) ID() string { return "AUTO-IDEMPOTENT-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (IdempotentWrite[T, V, Obs]) REQID() string { return "" }

// Check verifies that two identical Writes produce the same Observe.
//
// The law mutates state (it writes twice). Idempotence is a write-
// side property; checking it requires writing — but the second
// Write must not change anything visible to Observe.
func (l IdempotentWrite[T, V, Obs]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "IdempotentWrite_value")
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	before := l.Observe(rt, sut)
	if err := l.Write(rt, sut, v); err != nil {
		return fmt.Errorf("IdempotentWrite: value %v: second write errored: %v", v, err)
	}
	after := l.Observe(rt, sut)
	if diff := cmp.Diff(before, after); diff != "" {
		return fmt.Errorf("IdempotentWrite: value %v: second write changed state (-before +after):\n%s", v, diff)
	}
	return nil
}

// WriteObservable verifies that a value written via Write is
// retrievable through the paired Reader under the value's key — the
// most basic writer contract. Auto-emitted for Writer-class methods
// with a paired Reader.
//
// KeyOf extracts the lookup key from the written value; Read fetches
// it back. A write that is silently dropped, or a read that returns
// a divergent value, fails the law.
type WriteObservable[T any, V any, K comparable] struct {
	Write  func(*rapid.T, T, V) error
	Read   func(*rapid.T, T, K) (V, error)
	Values *rapid.Generator[V]
	KeyOf  func(V) K
}

// ID returns the stable identifier for this law.
func (WriteObservable[T, V, K]) ID() string { return "AUTO-WRITE-OBSERVABLE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (WriteObservable[T, V, K]) REQID() string { return "" }

// Check writes a value and verifies the paired reader returns it
// under the value's key.
func (l WriteObservable[T, V, K]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "WriteObservable_value")
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	got, err := l.Read(rt, sut, l.KeyOf(v))
	if err != nil {
		return fmt.Errorf("WriteObservable: value %v: not observable via read: %v", v, err)
	}
	if diff := cmp.Diff(v, got); diff != "" {
		return fmt.Errorf("WriteObservable: value %v: written value not observable (-write +read):\n%s", v, diff)
	}
	return nil
}

// TamperEvident verifies that out-of-band modification of stored
// data is detectable after the fact: Verify passes on freshly
// written data and fails once the store is tampered with. Auto-
// emitted for Writer/Appender methods carrying
// //testkit:tamper-evident.
//
// Tamper corrupts the store's backing state without going through
// Write, returning false when tampering is not applicable (e.g., an
// empty store) so the law skips rather than false-fails. A store
// with no integrity mechanism passes Verify even after tampering and
// fails the law.
type TamperEvident[T any, V any] struct {
	Write  func(*rapid.T, T, V) error
	Tamper func(*rapid.T, T) bool
	Verify func(*rapid.T, T) error
	Values *rapid.Generator[V]
}

// ID returns the stable identifier for this law.
func (TamperEvident[T, V]) ID() string { return "AUTO-TAMPER-EVIDENT" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TamperEvident[T, V]) REQID() string { return "" }

// Check writes a value, confirms Verify passes, tampers, and
// confirms Verify then detects the modification.
func (l TamperEvident[T, V]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "TamperEvident_value")
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Verify(rt, sut); err != nil {
		return fmt.Errorf("law: TamperEvident Verify failed on intact data: %v", err)
	}
	if !l.Tamper(rt, sut) {
		return nil // tampering not applicable; nothing to detect
	}
	if err := l.Verify(rt, sut); err == nil {
		return errors.New("law: TamperEvident Verify did not detect tampering")
	}
	return nil
}

// XSSSafe verifies that markup passed through Render is neutralized:
// no script-capable tag-opener survives in the rendered output.
// Auto-emitted for Writer/Mutator methods carrying
// //testkit:xss-safe.
//
// Render stores a payload and returns its HTML-rendered form. The
// law draws XSS vectors and fails if any token in Dangerous (default
// [defaultXSSTokens]) appears verbatim in the output — a correct
// escaper converts the leading '<' so the tag-opener cannot appear.
type XSSSafe[T any] struct {
	Render    func(rt *rapid.T, sut T, raw string) (string, error)
	Payloads  *rapid.Generator[string]
	Dangerous []string
}

// ID returns the stable identifier for this law.
func (XSSSafe[T]) ID() string { return "AUTO-XSS-SAFE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (XSSSafe[T]) REQID() string { return "" }

// Check renders an XSS vector and fails if a dangerous tag-opener
// survives unescaped in the output.
func (l XSSSafe[T]) Check(rt *rapid.T, sut, _ T) error {
	raw := l.Payloads.Draw(rt, "XSSSafe_payload")
	out, err := l.Render(rt, sut, raw)
	if err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	danger := l.Dangerous
	if len(danger) == 0 {
		danger = defaultXSSTokens
	}
	low := strings.ToLower(out)
	for _, tok := range danger {
		if strings.Contains(low, tok) {
			return fmt.Errorf("XSSSafe: rendered output contains unescaped %q (from vector %q): %q", tok, raw, out)
		}
	}
	return nil
}

// InjectionSafe verifies that a payload carrying SQL/shell
// metacharacters is treated as literal data: it round-trips through
// storage unchanged, and a separately-stored canary value is not
// disturbed by it. Auto-emitted for Writer/Mutator methods carrying
// //testkit:injection-safe.
//
// The law seeds CanaryKey with CanaryValue, stores an injection
// vector under a distinct key, then asserts both that the vector
// loads back verbatim (parameterization preserved it) and that the
// canary is intact (the injection did not break out to affect other
// data). A store that concatenates input into a command corrupts one
// or the other and fails.
type InjectionSafe[T any] struct {
	Store       func(rt *rapid.T, sut T, key, value string) error
	Load        func(rt *rapid.T, sut T, key string) (string, error)
	Payloads    *rapid.Generator[string]
	CanaryKey   string
	CanaryValue string
}

// ID returns the stable identifier for this law.
func (InjectionSafe[T]) ID() string { return "AUTO-INJECTION-SAFE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (InjectionSafe[T]) REQID() string { return "" }

// Check stores a canary and an injection vector, then verifies the
// vector round-trips verbatim and the canary is undisturbed.
func (l InjectionSafe[T]) Check(rt *rapid.T, sut, _ T) error {
	const target = "injectionsafe_target"
	if err := l.Store(rt, sut, l.CanaryKey, l.CanaryValue); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	payload := l.Payloads.Draw(rt, "InjectionSafe_payload")
	if err := l.Store(rt, sut, target, payload); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	got, err := l.Load(rt, sut, target)
	if err != nil {
		return fmt.Errorf("InjectionSafe: payload not retrievable: %v", err)
	}
	if got != payload {
		return fmt.Errorf("InjectionSafe: payload altered in storage (want %q, got %q)", payload, got)
	}
	canary, err := l.Load(rt, sut, l.CanaryKey)
	if err != nil {
		return fmt.Errorf("InjectionSafe: canary lost after storing payload %q: %v", payload, err)
	}
	if canary != l.CanaryValue {
		return fmt.Errorf(
			"InjectionSafe: payload %q corrupted canary (want %q, got %q)",
			payload,
			l.CanaryValue,
			canary,
		)
	}
	return nil
}

// CommutativeWrite verifies a;b == b;a observationally over the
// supplied Observe function. Auto-emitted for Mutator/Writer
// methods carrying //testkit:commutative.
//
// The law runs the pair-of-writes on two fresh impls — a;b on one,
// b;a on the other — using the consumer-supplied factory to
// construct them. The result Obs must agree.
type CommutativeWrite[T any, V any, Obs any] struct {
	Factory func() T
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (CommutativeWrite[T, V, Obs]) ID() string { return "AUTO-COMMUTATIVE-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (CommutativeWrite[T, V, Obs]) REQID() string { return "" }

// Check runs a;b on one impl and b;a on another, comparing Observe.
func (l CommutativeWrite[T, V, Obs]) Check(rt *rapid.T, _, _ T) error {
	a := l.Values.Draw(rt, "CommutativeWrite_a")
	b := l.Values.Draw(rt, "CommutativeWrite_b")

	ab := l.Factory()
	if err := l.Write(rt, ab, a); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Write(rt, ab, b); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}

	ba := l.Factory()
	if err := l.Write(rt, ba, b); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	if err := l.Write(rt, ba, a); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}

	obsAB := l.Observe(rt, ab)
	obsBA := l.Observe(rt, ba)
	if diff := cmp.Diff(obsAB, obsBA); diff != "" {
		return fmt.Errorf("CommutativeWrite: a=%v b=%v: order matters (-ab +ba):\n%s", a, b, diff)
	}
	return nil
}

// AtomicWrite verifies that a Writer returning an error leaves the
// observable state unchanged — error implies no partial mutation.
// Auto-emitted for Writers carrying //testkit:atomic.
//
// The law snapshots observable state via Observe, calls Write, and
// when Write errors compares the post-error snapshot against the
// pre-call snapshot. Successful writes are skipped (their
// mutation is checked by other laws).
type AtomicWrite[T any, V any, Obs any] struct {
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) Obs
}

// ID returns the stable identifier for this law.
func (AtomicWrite[T, V, Obs]) ID() string { return "AUTO-ATOMIC-WRITE" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (AtomicWrite[T, V, Obs]) REQID() string { return "" }

// Check verifies that errored writes leave state unchanged.
func (l AtomicWrite[T, V, Obs]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "AtomicWrite_value")
	before := l.Observe(rt, sut)
	err := l.Write(rt, sut, v)
	if err == nil {
		return nil
	}
	after := l.Observe(rt, sut)
	if diff := cmp.Diff(before, after); diff != "" {
		return fmt.Errorf("AtomicWrite: errored write mutated state (-before +after):\n%s", diff)
	}
	return nil
}

// ValidTransition verifies that Write only advances the named field
// through transitions allowed by a state-machine graph. Auto-
// emitted for Mutator/Writer methods carrying
// //testkit:valid-transition-only <Field>.
//
// The law consults the Allowed predicate to decide whether the
// observed before→after transition was legal; it does not enforce
// the rejection itself (the SUT must reject illegal writes on its
// own). The law only flags an after-state that the predicate
// declares invalid.
type ValidTransition[T any, V any, S comparable] struct {
	Write   func(*rapid.T, T, V) error
	Values  *rapid.Generator[V]
	Observe func(*rapid.T, T) S
	Allowed func(from, to S) bool
}

// ID returns the stable identifier for this law.
func (ValidTransition[T, V, S]) ID() string { return "AUTO-VALID-TRANSITION" }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (ValidTransition[T, V, S]) REQID() string { return "" }

// Check verifies any post-write state was reachable from the prior.
func (l ValidTransition[T, V, S]) Check(rt *rapid.T, sut, _ T) error {
	v := l.Values.Draw(rt, "ValidTransition_value")
	before := l.Observe(rt, sut)
	if err := l.Write(rt, sut, v); err != nil {
		return nil //nolint:nilerr // precondition failed; law vacuously holds
	}
	after := l.Observe(rt, sut)
	if before == after {
		return nil
	}
	if !l.Allowed(before, after) {
		return fmt.Errorf("ValidTransition: value %v: illegal %v → %v", v, before, after)
	}
	return nil
}
