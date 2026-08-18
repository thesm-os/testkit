// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"errors"
	"testing"
	"time"
)

// The signature-check primitives. A generated assertion is one line of
// method wiring over one of these; the semantic — what "reports a
// cancelled context" MEANS, and what the failure says — has exactly one
// home here. Before this file, every generated package restated these
// bodies once per method, which is the drift class the whole platform
// exists to kill: reword one message and eleven siblings go stale.
//
// Each primitive takes the method's name for the failure message and a
// closure that calls the method with the context the primitive supplies.
//
// # Why these do not delegate to the root testkit contract asserts
//
// The root's AssertCtxCancellation family is deliberately lax — it
// accepts Canceled or DeadlineExceeded interchangeably, because a
// hand-written test often cannot know which boundary fired first. These
// primitives are strict: //testkit:ctx is an EXPLICIT claim the interface
// author made, cancel and deadline are separate check families with
// separate IDs in the lock, and a suite that accepted either error for
// both families could not tell a subject that conflates them from one
// that honours them. Whether the strict method-aware forms move upstream
// and deprecate the lax four is a decision for the generator era; until
// then this paragraph is the fork's documentation.

// Survives asserts the call returns rather than panicking — the weakest
// check and the one that catches the most, because a method that panics
// on an ordinary value is one no other check reaches.
func Survives(tb testing.TB, method string, call func(ctx context.Context)) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Fatalf("%s panicked on a derived value (%v); supply one it accepts "+
				"through the suite config's pools", method, r)
		}
	}()
	call(tb.Context())
}

// ReportsCancelled asserts the method reports a context cancelled before
// the call as context.Canceled.
func ReportsCancelled(tb testing.TB, method string, call func(ctx context.Context) error) {
	tb.Helper()
	ctx, cancel := context.WithCancel(tb.Context())
	cancel()
	if err := call(ctx); !errors.Is(err, context.Canceled) {
		tb.Errorf("%s must report a cancelled context: got %v, want %v",
			method, err, context.Canceled)
	}
}

// ReportsDeadlineExceeded asserts the method reports an already-expired
// deadline as context.DeadlineExceeded.
func ReportsDeadlineExceeded(tb testing.TB, method string, call func(ctx context.Context) error) {
	tb.Helper()
	ctx, cancel := context.WithDeadline(tb.Context(), time.Now().Add(-time.Hour))
	defer cancel()
	if err := call(ctx); !errors.Is(err, context.DeadlineExceeded) {
		tb.Errorf("%s must report an expired deadline: got %v, want %v",
			method, err, context.DeadlineExceeded)
	}
}

// ToleratesNilContext asserts the method returns an error rather than
// panicking when handed the nil context an errant caller will eventually
// supply. Returning an error is a failed request; dereferencing nil is an
// outage.
func ToleratesNilContext(tb testing.TB, method string, call func(ctx context.Context) error) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Errorf("%s panicked on a nil context (%v); return an error instead", method, r)
		}
	}()
	//nolint:staticcheck // passing nil is the point of this check
	if err := call(nil); err == nil {
		// The recorded claim is "returns an error", and an assertion
		// weaker than its claim is a silent green: accepting nil and
		// working uncancellably is exactly what the claim rules out.
		tb.Errorf("%s returned nil on a nil context; return an error instead", method)
	}
}
