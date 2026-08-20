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

// What each primitive's failure reads, as the substring a proof holds it
// to. Declared beside the messages rather than restated in the generator
// that plants the defects: a proof asserting a red it cannot see is a
// proof that counts an incidental failure as evidence, and the way that
// happens is somebody rewording a message here without knowing the
// generator quoted it.
//
// The generator emits these into [prove.Defect.Reason], so a reworded
// message and a stale quote become a compile error rather than a proof
// that silently stops proving.
const (
	// RedPanicked is what [Survives] reports.
	RedPanicked = "panicked on a derived value"

	// RedCancelled is what [ReportsCancelled] reports.
	RedCancelled = "must report a cancelled context"

	// RedDeadline is what [ReportsDeadlineExceeded] reports.
	RedDeadline = "must report an expired deadline"

	// RedNilContext is what [ToleratesNilContext] reports for a subject
	// that answers, which is the arm a planted defect reaches. The other
	// arm — a subject that panics — is a different failure and is not
	// what any defect here plants.
	RedNilContext = "returned nil on a nil context"

	// RedNilArgument is what [ToleratesNilArgument] reports for a
	// subject that answers, the arm a planted defect reaches — the
	// other, a subject that panics, is a different failure and is not
	// what any defect here plants.
	RedNilArgument = "returned nil for a nil argument"
)

// The names the generator spells when it emits a call to one of the
// guards below, declared here for the reason the reds above are: a
// generated body names the function in text, so a rename here that
// misses the generator produces a consumer file calling something that
// no longer exists. The generator holds one of these rather than a
// literal, and TestGuardNamesAreTheFunctions holds each to the function
// it claims to name.
const (
	// GuardCancelled names [ReportsCancelled].
	GuardCancelled = "ReportsCancelled"

	// GuardDeadline names [ReportsDeadlineExceeded].
	GuardDeadline = "ReportsDeadlineExceeded"

	// GuardNilContext names [ToleratesNilContext].
	GuardNilContext = "ToleratesNilContext"

	// GuardNilArgument names [ToleratesNilArgument].
	GuardNilArgument = "ToleratesNilArgument"
)

// Survives asserts the call returns rather than panicking — the weakest
// check and the one that catches the most, because a method that panics
// on an ordinary value is one no other check reaches.
func Survives(tb testing.TB, method string, call func(ctx context.Context)) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Fatalf("%s "+RedPanicked+" (%v); supply one it accepts "+
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
		tb.Errorf("%s "+RedCancelled+": got %v, want %v",
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
		tb.Errorf("%s "+RedDeadline+": got %v, want %v",
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
		tb.Errorf("%s "+RedNilContext+"; return an error instead", method)
	}
}

// ToleratesNilArgument asserts the method returns an error rather than
// panicking when one of its nilable arguments arrives nil.
//
// The context is real here and the nil is in the call, which is the
// whole difference from [ToleratesNilContext]: the caller's context is
// the harness's to supply, and which argument is nil is the generated
// body's to spell. So this takes the same closure and only judges what
// comes back.
//
// A nil pointer, slice or map reaching a method is an ordinary caller
// mistake and a defined refusal is the whole claim. Dereferencing it is
// an outage in the caller's process, which is the failure this exists to
// catch.
func ToleratesNilArgument(tb testing.TB, method string, call func(ctx context.Context) error) {
	tb.Helper()
	defer func() {
		if r := recover(); r != nil {
			tb.Errorf("%s panicked on a nil argument (%v); return an error instead", method, r)
		}
	}()
	if err := call(tb.Context()); err == nil {
		// The recorded claim is "returns an error", and an assertion
		// weaker than its claim is a silent green: accepting nil and
		// carrying on is exactly what the claim rules out.
		tb.Errorf("%s "+RedNilArgument+"; return an error instead", method)
	}
}
